def test_invalid_handle(C):
    assert C.SQLColumns(C.NULL, C.NULL, 0, C.NULL, 0, C.NULL, 0, C.NULL, 0) == C.SQL_INVALID_HANDLE
