def test_invalid_handle(C):
    assert C.SQLRowCount(C.NULL, C.NULL) == C.SQL_INVALID_HANDLE


def test_no_data(C, stmt_handle):
    length = C.ffi.new("SQLLEN*")
    length[0] = 1

    assert C.SQLRowCount(stmt_handle, length) == C.SQL_SUCCESS
    assert length[0] == 0


def test_data(C, stmt_handle):
    table = C.ffi.new("char[]", b"TableName")
    # This happens to return two columns at the moment
    C.SQLColumns(stmt_handle, C.NULL, 0, C.NULL, 0, table, len(table), C.NULL, 0)

    length = C.ffi.new("SQLLEN*")
    length[0] = 1

    assert C.SQLRowCount(stmt_handle, length) == C.SQL_SUCCESS
    assert length[0] == 2
