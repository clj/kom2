def test_invalid_handle(C):
    assert C.SQLDescribeCol(C.NULL, 0, C.NULL, 0, C.NULL, C.NULL, C.NULL, C.NULL, C.NULL) == C.SQL_INVALID_HANDLE
