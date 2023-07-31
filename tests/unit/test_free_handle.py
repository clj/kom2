import pytest


def test_free_invalid_type(C):
    assert C.SQLFreeHandle(9999, C.NULL) == C.SQL_INVALID_HANDLE


@pytest.mark.parametrize(
    "handle_type",
    [
        "SQL_HANDLE_ENV",
        "SQL_HANDLE_DBC",
        "SQL_HANDLE_STMT",
        "SQL_HANDLE_DESC",
    ],
)
def test_free_invalid_handle(C, handle_type):
    assert C.SQLFreeHandle(getattr(C, handle_type), C.NULL) == C.SQL_INVALID_HANDLE
