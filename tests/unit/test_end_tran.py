import pytest

@pytest.mark.parametrize("completion_type", [
    "SQL_COMMIT",
    "SQL_ROLLBACK"
])
def test_allways_success(C, completion_type):
    C.SQLEndTran(0, C.NULL, getattr(C, completion_type)) == C.SQL_SUCCESS