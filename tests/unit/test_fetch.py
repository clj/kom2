def test_invalid_handle(C):
    assert C.SQLFetch(C.NULL) == C.SQL_INVALID_HANDLE
