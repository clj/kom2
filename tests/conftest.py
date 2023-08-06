import io
import os
import platform
import socket
import subprocess
import sys

import pycparser
import pypyodbc
import pytest
from cffi import FFI
from pcpp import Action
from pcpp import OutputDirective
from pcpp import Preprocessor
from pycparser import c_ast
from pycparser.c_generator import CGenerator
from pycparser.c_parser import CParser


@pytest.fixture
def port():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind(("localhost", 0))
    yield s.getsockname()
    s.close()


@pytest.fixture(scope="session")
def driver_library():
    if name := os.getenv("KOM2_DRIVER_LIBRARY"):
        return name
    return {
        "Linux": "kom2.so",
        "Darwin": "kom2.dylib",
        "Windows": os.path.abspath("kom2.dll"),
    }[platform.system()]


def get_driver_name():
    if name := os.getenv("KOM2_DRIVER_NAME"):
        return name
    if platform.system() == "Windows":
        return "kom2"
    return None


@pytest.fixture(scope="session")
def driver_name(driver_library):
    if driver_name := get_driver_name():
        return driver_name
    return driver_library


@pytest.fixture(scope="session")
def platform_defines():
    result = subprocess.run(["gcc", "-dM", "-E", "-"], input=b"", capture_output=True)
    return result.stdout.decode("utf-8")


class MyPreprocessor(Preprocessor):
    def on_include_not_found(
        self, is_malformed, is_system_include, curdir, includepath
    ):
        if includepath == "mm_malloc.h" or includepath.endswith("intrin.h"):
            raise OutputDirective(action=Action.IgnoreAndRemove)

        super().on_include_not_found(
            is_malformed, is_system_include, curdir, includepath
        )


class MyCGenerator(CGenerator):
    def __init__(self, compound_typedefs=[], **kwargs):
        self.compound_typedefs = compound_typedefs
        self.compound_typedef_types = {}
        for name, nodes in self.compound_typedefs.items():
            for node in nodes:
                self.compound_typedef_types[node.name] = name
        super().__init__(**kwargs)

    def visit_Typedef(self, n):
        if (
            n.name in self.compound_typedef_types
            and self.compound_typedefs[self.compound_typedef_types[n.name]].index(n)
            != 0
        ):
            return ""

        return super().visit_Typedef(n)

    def _generate_type(self, n, modifiers=[], emit_declname=True, only_declname=False):
        """Recursive generation from a type node. n is the type node.
        modifiers collects the PtrDecl, ArrayDecl and FuncDecl modifiers
        encountered on the way down to a TypeDecl, to allow proper
        generation from it.
        """
        typ = type(n)
        # ~ print(n, modifiers)

        if typ == c_ast.TypeDecl:
            s = ""
            nstrs = []
            if not only_declname:
                if n.quals:
                    s += " ".join(n.quals) + " "
                s += self.visit(n.type)

                if t := self.compound_typedef_types.get(n.declname):
                    nstrs = [
                        self._generate_type(
                            node.type, modifiers, emit_declname, only_declname=True
                        )
                        for node in self.compound_typedefs[t][1:]
                    ]

            nstr = n.declname if n.declname and emit_declname else ""
            # Resolve modifiers.
            # Wrap in parens to distinguish pointer to array and pointer to
            # function syntax.
            #
            for i, modifier in enumerate(modifiers):
                if isinstance(modifier, c_ast.ArrayDecl):
                    if i != 0 and isinstance(modifiers[i - 1], c_ast.PtrDecl):
                        nstr = "(" + nstr + ")"
                    nstr += "["
                    if modifier.dim_quals:
                        nstr += " ".join(modifier.dim_quals) + " "
                    nstr += self.visit(modifier.dim) + "]"
                elif isinstance(modifier, c_ast.FuncDecl):
                    if i != 0 and isinstance(modifiers[i - 1], c_ast.PtrDecl):
                        nstr = "(" + nstr + ")"
                    nstr += "(" + self.visit(modifier.args) + ")"
                elif isinstance(modifier, c_ast.PtrDecl):
                    if modifier.quals:
                        nstr = "* %s%s" % (
                            " ".join(modifier.quals),
                            " " + nstr if nstr else "",
                        )
                    else:
                        nstr = "*" + nstr
            if nstr:
                nstrs = [nstr] + nstrs
            if nstrs:
                s += " " + ", ".join(nstrs)
            return s
        elif typ == c_ast.Decl:
            return self._generate_decl(n.type, only_declname=only_declname)
        elif typ == c_ast.Typename:
            return self._generate_type(
                n.type, emit_declname=emit_declname, only_declname=only_declname
            )
        elif typ == c_ast.IdentifierType:
            return " ".join(n.names) + " "
        elif typ in (c_ast.ArrayDecl, c_ast.PtrDecl, c_ast.FuncDecl):
            return self._generate_type(
                n.type,
                modifiers + [n],
                emit_declname=emit_declname,
                only_declname=only_declname,
            )
        else:
            return self.visit(n)


class ModificationsVisitor(pycparser.c_ast.NodeVisitor):
    functions = []
    typedef_structs = {}

    def visit_FuncDef(self, node):
        self.functions.append(node)

    # See: https://github.com/eliben/pycparser/issues/195#issuecomment-373539111
    def visit_Typedef(self, node):
        type_decl = node.type
        if not isinstance(type_decl, pycparser.c_ast.TypeDecl):
            type_decl = type_decl.type
        if not (
            isinstance(type_decl, pycparser.c_ast.TypeDecl)
            and (
                isinstance(type_decl.type, pycparser.c_ast.Struct)
                or isinstance(type_decl.type, pycparser.c_ast.Enum)
            )
        ):
            return
        if (
            isinstance(type_decl.type, pycparser.c_ast.Struct)
            and not type_decl.type.decls
        ):
            return
        if not type_decl.type.name:
            return
        self.typedef_structs.setdefault(type_decl.type.name, []).append(node)


class CLibrary:
    def __init__(
        self,
        input,
        library_name,
        /,
        include_paths=[
            "/usr/include",
            "/usr/local/include",
            "C:/ProgramData/chocolatey/lib/mingw/tools/install/mingw64/x86_64-w64-mingw32/include",
        ],
        defines=[],
    ):
        self.p = MyPreprocessor()
        for path in include_paths:
            self.p.add_path(path)
        for define in defines:
            self.p.define(define)
        self.p.parse(input)

        buf = io.StringIO()
        self.p.write(buf)

        output = buf.getvalue()

        parser = CParser()
        ast = parser.parse(output)

        visitor = ModificationsVisitor()
        visitor.visit(ast)

        # Remove all function definitions, cffi doesn't like them
        for node in visitor.functions:
            ast.ext.remove(node)

        # Generate code without duplicate struct typedefs
        # https://github.com/eliben/pycparser/issues/210#issuecomment-330494327
        generator = MyCGenerator(compound_typedefs=visitor.typedef_structs)
        self.output = generator.visit(ast)

        ffi = FFI()
        ffi.cdef(self.output, override=True)
        self.c = ffi.dlopen(library_name)
        self.ffi = ffi

    def __getattr__(self, name):
        if name == "NULL":
            return self.ffi.NULL

        try:
            return getattr(self.c, name)
        except AttributeError:
            return self.p.evalexpr(list(self.p.parsegen(name)))[0]


# Getting a working header file for windows is complicated, by:
# See: https://github.com/eliben/pycparser/issues/210#issuecomment-330494327
# and by the fact that including windows.h takes a very long time to process
# see also:
# and: https://github.com/eliben/pycparser/blob/6cf69df2/utils/fake_libc_include/_fake_typedefs.h#L7
@pytest.fixture(scope="session")
def C(driver_library, platform_defines):
    return CLibrary(
        platform_defines
        + """
        #if defined(_WIN32)
        # define __asm__(...)
        # define __attribute__(x)
        # define __extension__
        # define __const const
        # define __inline__ inline
        # define __inline inline
        # define __restrict
        # define __restrict__
        # define __signed__ signed
        # define __GNUC_VA_LIST
        # define __gnuc_va_list char
        # define __volatile__(...)
        # define __builtin_offsetof(x, y) 0
        # define __has_builtin(x) 1
        # define NT_INCLUDED
        # include <_mingw.h>
        # include <basetsd.h>
        # define HWND void*
        # define HANDLE void*
        # define DECLARE_HANDLE(name) typedef HANDLE name
        # include <minwindef.h>
        # include <mapinls.h>
        # include <guiddef.h>
        # define VOID void
        #endif
        #include <sqltypes.h>
        #include <sql.h>
        #include <sqlext.h>
        """,
        driver_library,
    )


@pytest.fixture
def env_handle(C):
    with C.ffi.new("SQLHANDLE*") as handle:
        C.SQLAllocHandle(C.SQL_HANDLE_ENV, C.NULL, handle)
        yield handle[0]


@pytest.fixture
def conn_handle(C, env_handle):
    with C.ffi.new("SQLHANDLE*") as handle:
        C.SQLAllocHandle(C.SQL_HANDLE_DBC, env_handle, handle)
        yield handle[0]


@pytest.fixture
def stmt_handle(C, conn_handle):
    with C.ffi.new("SQLHANDLE*") as handle:
        C.SQLAllocHandle(C.SQL_HANDLE_STMT, conn_handle, handle)
        yield handle[0]


def maybe_skip_windows():
    if not sys.platform.startswith("win"):
        return False
    if not (name := get_driver_name()):
        return False
    try:
        pypyodbc.connect(f"Driver={name}")
    except pypyodbc.Error as error:
        return error.args[0] == "IM002"
    return False
