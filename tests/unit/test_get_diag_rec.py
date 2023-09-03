import pytest


def test_invalid_handle(C):
    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_STMT, C.NULL, 0, C.NULL, C.NULL, C.NULL, 0, C.NULL
    )
    assert result == C.SQL_INVALID_HANDLE


def test_invalid_handle_type(C, env_handle):
    result = C.SQLGetDiagRec(9999, env_handle, 0, C.NULL, C.NULL, C.NULL, 0, C.NULL)
    assert result == C.SQL_INVALID_HANDLE


def test_mismatch_handle_type(C, env_handle):
    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_STMT, env_handle, 0, C.NULL, C.NULL, C.NULL, 0, C.NULL
    )
    assert result == C.SQL_INVALID_HANDLE


def test_no_error(C, env_handle):
    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_ENV, env_handle, 0, C.NULL, C.NULL, C.NULL, 0, C.NULL
    )
    assert result == C.SQL_NO_DATA


def test_no_error_zeroed(C, env_handle):
    sql_state = C.ffi.new("SQLCHAR[]", 6)
    native_error = C.ffi.new("SQLINTEGER*")
    buffer = C.ffi.new("SQLCHAR[]", 100)
    text_len = C.ffi.new("SQLSMALLINT*")

    native_error[0] = 1

    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_ENV,
        env_handle,
        0,
        sql_state,
        native_error,
        buffer,
        len(buffer),
        text_len,
    )
    assert result == C.SQL_NO_DATA
    assert C.ffi.string(sql_state) == b""
    assert native_error[0] == 0
    assert C.ffi.string(buffer) == b""
    assert text_len[0] == 0


def test_env_error(C, env_handle):
    assert (
        C.SQLSetEnvAttr(env_handle, C.SQL_ATTR_CONNECTION_POOLING, C.NULL, 0)
        == C.SQL_ERROR
    )

    sql_state = C.ffi.new("SQLCHAR[]", 6)
    buffer = C.ffi.new("SQLCHAR[]", 100)
    text_len = C.ffi.new("SQLSMALLINT*")
    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_ENV,
        env_handle,
        1,
        sql_state,
        C.NULL,
        buffer,
        len(buffer),
        text_len,
    )
    assert result == C.SQL_SUCCESS
    assert C.ffi.string(sql_state) == b"HYC00"
    assert C.ffi.buffer(sql_state, 6)[:] == b"HYC00\x00"
    assert C.ffi.string(buffer) == b"Unsupported attribute"
    assert text_len[0] == len(C.ffi.string(buffer))


def test_env_error_repeated(C, env_handle):
    assert (
        C.SQLSetEnvAttr(env_handle, C.SQL_ATTR_CONNECTION_POOLING, C.NULL, 0)
        == C.SQL_ERROR
    )

    sql_state = C.ffi.new("SQLCHAR[]", 6)
    buffer = C.ffi.new("SQLCHAR[]", 100)
    text_len = C.ffi.new("SQLSMALLINT*")
    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_ENV,
        env_handle,
        1,
        sql_state,
        C.NULL,
        buffer,
        len(buffer),
        text_len,
    )
    assert result == C.SQL_SUCCESS
    assert C.ffi.string(sql_state) == b"HYC00"
    assert C.ffi.string(buffer) == b"Unsupported attribute"
    assert text_len[0] == len(C.ffi.string(buffer))

    sql_state = C.ffi.new("SQLCHAR[]", 6)
    buffer = C.ffi.new("SQLCHAR[]", 100)
    text_len = C.ffi.new("SQLSMALLINT*")
    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_ENV,
        env_handle,
        1,
        sql_state,
        C.NULL,
        buffer,
        len(buffer),
        text_len,
    )
    assert result == C.SQL_SUCCESS
    assert C.ffi.string(sql_state) == b"HYC00"
    assert C.ffi.string(buffer) == b"Unsupported attribute"
    assert text_len[0] == len(C.ffi.string(buffer))


@pytest.mark.parametrize(
    "rec_number, expected",
    [
        (-10, "SQL_ERROR"),
        (-1, "SQL_ERROR"),
        (0, "SQL_ERROR"),
        (1, "SQL_SUCCESS"),
        (2, "SQL_NO_DATA"),
        (3, "SQL_NO_DATA"),
        (10, "SQL_NO_DATA"),
        (100, "SQL_NO_DATA"),
    ],
)
def test_env_error_other_records(C, env_handle, rec_number, expected):
    assert (
        C.SQLSetEnvAttr(env_handle, C.SQL_ATTR_CONNECTION_POOLING, C.NULL, 0)
        == C.SQL_ERROR
    )

    sql_state = C.ffi.new("SQLCHAR[]", 6)
    buffer = C.ffi.new("SQLCHAR[]", 100)
    text_len = C.ffi.new("SQLSMALLINT*")
    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_ENV,
        env_handle,
        rec_number,
        sql_state,
        C.NULL,
        buffer,
        len(buffer),
        text_len,
    )
    assert result == getattr(C, expected)


def test_env_null_message_text(C, env_handle):
    assert (
        C.SQLSetEnvAttr(env_handle, C.SQL_ATTR_CONNECTION_POOLING, C.NULL, 0)
        == C.SQL_ERROR
    )

    sql_state = C.ffi.new("SQLCHAR[]", 6)
    text_len = C.ffi.new("SQLSMALLINT*")
    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_ENV,
        env_handle,
        1,
        sql_state,
        C.NULL,
        C.NULL,
        0,
        text_len,
    )
    assert result == C.SQL_SUCCESS
    assert C.ffi.string(sql_state) == b"HYC00"
    assert text_len[0] == len(b"Unsupported attribute")


def test_env_truncated_message_text(C, env_handle):
    assert (
        C.SQLSetEnvAttr(env_handle, C.SQL_ATTR_CONNECTION_POOLING, C.NULL, 0)
        == C.SQL_ERROR
    )

    sql_state = C.ffi.new("SQLCHAR[]", 6)
    buffer = C.ffi.new("SQLCHAR[]", 5)
    text_len = C.ffi.new("SQLSMALLINT*")
    result = C.SQLGetDiagRec(
        C.SQL_HANDLE_ENV,
        env_handle,
        1,
        sql_state,
        C.NULL,
        buffer,
        len(buffer),
        text_len,
    )
    assert result == C.SQL_SUCCESS
    assert C.ffi.string(sql_state) == b"HYC00"
    assert C.ffi.string(buffer) == b"Unsu"
    assert text_len[0] == len(b"Unsupported attribute")
