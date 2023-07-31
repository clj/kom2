import pytest


def test_alloc_env(C):
    handle = C.ffi.new("SQLHANDLE*")
    assert handle[0] == C.NULL
    assert C.SQLAllocHandle(C.SQL_HANDLE_ENV, C.NULL, handle) == C.SQL_SUCCESS
    assert handle[0] != C.NULL
    assert handle[0] != C.SQL_NULL_HENV


def test_alloc_dbc(C, env_handle):
    handle = C.ffi.new("SQLHANDLE*")
    assert handle[0] == C.NULL
    assert C.SQLAllocHandle(C.SQL_HANDLE_DBC, env_handle, handle) == C.SQL_SUCCESS
    assert handle[0] != C.NULL
    assert handle[0] != C.SQL_NULL_HDBC


def test_alloc_stmt(C, conn_handle):
    handle = C.ffi.new("SQLHANDLE*")
    assert handle[0] == C.NULL
    assert C.SQLAllocHandle(C.SQL_HANDLE_STMT, conn_handle, handle) == C.SQL_SUCCESS
    assert handle[0] != C.NULL
    assert handle[0] != C.SQL_NULL_HSTMT


def test_alloc_invalid(C):
    handle = C.ffi.new("SQLHANDLE*")
    assert handle[0] == C.NULL
    assert C.SQLAllocHandle(9999, C.NULL, handle) == C.SQL_ERROR
    assert handle[0] == C.NULL


@pytest.mark.parametrize(
    "handle_type, input_handle, error_handle",
    [
        ("SQL_HANDLE_DBC", pytest.lazy_fixture("conn_handle"), "SQL_NULL_HDBC"),
        ("SQL_HANDLE_DBC", pytest.lazy_fixture("stmt_handle"), "SQL_NULL_HDBC"),
        ("SQL_HANDLE_STMT", pytest.lazy_fixture("env_handle"), "SQL_NULL_HSTMT"),
        ("SQL_HANDLE_STMT", pytest.lazy_fixture("stmt_handle"), "SQL_NULL_HSTMT"),
    ],
)
def test_alloc_invalid_handle(C, handle_type, input_handle, error_handle):
    handle_type = getattr(C, handle_type)
    error_handle = getattr(C, error_handle)
    handle = C.ffi.new("SQLHANDLE*")
    assert handle[0] == C.NULL
    assert C.SQLAllocHandle(handle_type, input_handle, handle) == C.SQL_ERROR
    assert handle[0] == C.ffi.cast("SQLHANDLE", error_handle)
