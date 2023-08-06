import pytest


@pytest.fixture
def get_diag_rec(C, stmt_handle):
    def fn():
        sql_state = C.ffi.new("SQLCHAR[]", 6)
        buffer = C.ffi.new("SQLCHAR[]", 1000)
        text_len = C.ffi.new("SQLSMALLINT*")
        result = C.SQLGetDiagRec(
            C.SQL_HANDLE_STMT,
            stmt_handle,
            1,
            sql_state,
            C.NULL,
            buffer,
            len(buffer),
            text_len,
        )
        assert result == C.SQL_SUCCESS
        return sql_state, result

    return fn


def test_invalid_handle(C):
    assert (
        C.SQLDescribeParam(C.NULL, 2, C.NULL, C.NULL, C.NULL, C.NULL)
        == C.SQL_INVALID_HANDLE
    )


def test_param_one(C, stmt_handle):
    data_type_ptr = C.ffi.new("SQLSMALLINT *")
    parameter_size_ptr = C.ffi.new("SQLULEN *")
    decimal_digits_ptr = C.NULL
    nullable_ptr = C.ffi.new("SQLSMALLINT *")

    assert (
        C.SQLDescribeParam(
            stmt_handle,
            1,
            data_type_ptr,
            parameter_size_ptr,
            decimal_digits_ptr,
            nullable_ptr,
        )
        == C.SQL_SUCCESS
    )
    assert data_type_ptr[0] == C.SQL_VARCHAR
    # assert parameter_size_ptr[0] == C.SQL_NO_TOTAL    Not sure this is correct
    assert nullable_ptr[0] == C.SQL_NO_NULLS


def test_invalid_param(C, stmt_handle, get_diag_rec):
    assert (
        C.SQLDescribeParam(stmt_handle, 2, C.NULL, C.NULL, C.NULL, C.NULL)
        == C.SQL_ERROR
    )
    sql_state, _ = get_diag_rec()
    assert C.ffi.string(sql_state) == b"07009"
