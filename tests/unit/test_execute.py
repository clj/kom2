def test_invalid_handle(C):
    assert C.SQLExecute(C.NULL) == C.SQL_INVALID_HANDLE
