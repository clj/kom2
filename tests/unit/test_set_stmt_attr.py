def test_invalid_handle(C):
    assert C.SQLSetStmtAttr(C.NULL, 0, C.NULL, 0) == C.SQL_INVALID_HANDLE
