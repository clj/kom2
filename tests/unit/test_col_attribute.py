def test_invalid_handle(C):
    assert C.SQLColAttribute(C.NULL, 0, 0, C.NULL, 0, C.NULL, C.NULL) == C.SQL_INVALID_HANDLE
