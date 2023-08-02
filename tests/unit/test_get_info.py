def test_connect_invalid_handle(C):
    assert C.SQLGetInfo(C.NULL, 0, C.NULL, 0, C.NULL) == C.SQL_INVALID_HANDLE
