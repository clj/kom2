def test_connect_invalid_handle(C):
    assert C.SQLConnect(C.NULL, C.NULL, 0, C.NULL, 0, C.NULL, 0) == C.SQL_INVALID_HANDLE


def test_driver_connect_invalid_handle(C):
    assert C.SQLDriverConnect(C.NULL, C.NULL, C.NULL, 0, C.NULL, 0, C.NULL, 0) == C.SQL_INVALID_HANDLE
