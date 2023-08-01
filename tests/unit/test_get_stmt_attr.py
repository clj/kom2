def test_invalid_handle(C):
    assert C.SQLGetStmtAttr(C.NULL, 0, C.NULL, 0, C.NULL) == C.SQL_INVALID_HANDLE
