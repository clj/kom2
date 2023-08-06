import pytest


def test_invalid_handle(C):
    assert C.SQLGetStmtAttr(C.NULL, 0, C.NULL, 0, C.NULL) == C.SQL_INVALID_HANDLE


desc_attrs = [
    "SQL_ATTR_IMP_ROW_DESC",
    "SQL_ATTR_APP_ROW_DESC",
    "SQL_ATTR_IMP_PARAM_DESC",
    "SQL_ATTR_APP_PARAM_DESC",
]


@pytest.mark.parametrize("attr", desc_attrs)
def test_descs(C, stmt_handle, attr):
    buffer = C.ffi.new("SQLHDESC*")
    length = C.ffi.sizeof("SQLHDESC")
    length_ptr = C.ffi.new("SQLINTEGER*")

    attr = getattr(C, attr)
    assert (
        C.SQLGetStmtAttr(stmt_handle, attr, buffer, length, length_ptr) == C.SQL_SUCCESS
    )
    assert int(C.ffi.cast("unsigned int", buffer[0])) == 0xDEADBEEF
    assert length_ptr[0] == 8


@pytest.mark.parametrize("attr", desc_attrs)
def test_descs_no_length(C, stmt_handle, attr):
    buffer = C.ffi.new("SQLHDESC*")
    length = C.ffi.sizeof("SQLHDESC")

    attr = getattr(C, attr)
    assert C.SQLGetStmtAttr(stmt_handle, attr, buffer, length, C.NULL) == C.SQL_SUCCESS
    assert int(C.ffi.cast("unsigned int", buffer[0])) == 0xDEADBEEF
