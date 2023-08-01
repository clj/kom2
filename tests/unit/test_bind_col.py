def test_invalid_handle(C):
    assert C.SQLBindCol(C.NULL, 0, 0, C.NULL, 0, C.NULL) == C.SQL_INVALID_HANDLE
