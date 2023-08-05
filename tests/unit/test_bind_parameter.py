import pytest


def test_invalid_handle(C):
    assert C.SQLBindParameter(C.NULL, 0, 0, 0, 0, 0, 0, C.NULL, 0, C.NULL) == C.SQL_INVALID_HANDLE


def test_valid(C, stmt_handle):
    buffer = C.ffi.new("SQLCHAR[]", 100)
    assert C.SQLBindParameter(stmt_handle, 0, C.SQL_PARAM_INPUT, C.SQL_C_CHAR, 0, 0, 0, buffer, len(buffer), C.NULL) == C.SQL_SUCCESS


@pytest.mark.parametrize("input_output_type", ["SQL_PARAM_OUTPUT", "SQL_PARAM_INPUT_OUTPUT", "SQL_PARAM_INPUT_OUTPUT_STREAM", "SQL_PARAM_OUTPUT_STREAM"])
def test_invalid_input_output_types(C, stmt_handle, input_output_type):
    assert C.SQLBindParameter(stmt_handle, 0, getattr(C, input_output_type), 0, 0, 0, 0, C.NULL, 0, C.NULL) == C.SQL_ERROR


@pytest.mark.parametrize("value_type", ["SQL_C_WCHAR", "SQL_C_FLOAT", "SQL_C_SSHORT", "SQL_C_ULONG"])
def test_invalid_value_types(C, stmt_handle, value_type):
    assert C.SQLBindParameter(stmt_handle, 0, C.SQL_PARAM_INPUT, getattr(C, value_type), 0, 0, 0, C.NULL, 0, C.NULL) == C.SQL_ERROR
