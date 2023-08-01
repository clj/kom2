def test_invalid_handle(C):
    assert C.SQLFetchScroll(C.NULL, 0, 0) == C.SQL_INVALID_HANDLE
