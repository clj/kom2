def test_invalid_handle(C):
    assert C.SQLNumResultCols(C.NULL, C.NULL) == C.SQL_INVALID_HANDLE
