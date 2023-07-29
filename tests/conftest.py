import io
import os
import platform
import socket

from cffi import FFI
import pytest
from pcpp import Preprocessor


@pytest.fixture
def port():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind(("localhost", 0))
    yield s.getsockname()
    s.close()


@pytest.fixture(scope="session")
def driver_name():
    if name := os.getenv("KOM2_DRIVER_NAME"):
        return name
    return {"Linux": "kom2.so", "Darwin": "kom2.dylib", "Windows": "kom2.dll"}[
        platform.system()
    ]


class CLibrary:
    def __init__(
        self,
        input,
        library_name,
        include_paths=[
            "/usr/local/include",
        ],
    ):
        self.p = Preprocessor()
        for path in include_paths:
            self.p.add_path(path)
        self.p.parse(input)

        buf = io.StringIO()
        self.p.write(buf)

        self.output = buf.getvalue()
        ffi = FFI()
        ffi.cdef(self.output)
        self.c = ffi.dlopen(library_name)
        self.ffi = ffi

    def __getattr__(self, name):
        if name == "NULL":
            return self.ffi.NULL

        try:
            return getattr(self.c, name)
        except AttributeError:
            return self.p.evalexpr(list(self.p.parsegen(name)))[0]


@pytest.fixture(scope="session")
def C(driver_name):
    return CLibrary(
        """
        #include <sqltypes.h>
        #include <sql.h>
        #include <sqlext.h>
        """,
        driver_name,
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
