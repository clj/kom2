import pytest


def test_invalid_handle(C):
    result = C.SQLGetDiagField(C.SQL_HANDLE_STMT, C.NULL, 0, 0, C.NULL, 0, C.NULL)
    assert result == C.SQL_INVALID_HANDLE


def test_invalid_handle_type(C, env_handle):
    result = C.SQLGetDiagField(9999, env_handle, 0, 0, C.NULL, 0, C.NULL)
    assert result == C.SQL_INVALID_HANDLE


def test_mismatch_handle_type(C, env_handle):
    result = C.SQLGetDiagField(C.SQL_HANDLE_STMT, env_handle, 0, 0, C.NULL, 0, C.NULL)
    assert result == C.SQL_INVALID_HANDLE


def test_no_error(C, env_handle):
    result = C.SQLGetDiagField(C.SQL_HANDLE_ENV, env_handle, 0, 0, C.NULL, 0, C.NULL)
    assert result == C.SQL_NO_DATA


@pytest.fixture
def force_error(C, env_handle):
    assert (
        C.SQLSetEnvAttr(env_handle, C.SQL_ATTR_CONNECTION_POOLING, C.NULL, 0)
        == C.SQL_ERROR
    )


@pytest.mark.parametrize(
    "diag_identifier, expected",
    [
        ("SQL_DIAG_NUMBER", 1),
        ("SQL_DIAG_NATIVE", 0),
    ],
)
def test_error_int(C, force_error, env_handle, diag_identifier, expected):
    buffer = C.ffi.new("SQLINTEGER*")
    text_len = C.ffi.new("SQLSMALLINT*")
    result = C.SQLGetDiagField(
        C.SQL_HANDLE_ENV,
        env_handle,
        1,
        getattr(C, diag_identifier),
        buffer,
        C.SQL_IS_INTEGER,
        text_len,
    )
    assert result == C.SQL_SUCCESS
    assert buffer[0] == expected


@pytest.mark.parametrize(
    "diag_identifier, buffer_length, expected_str, expected_len, expected_ret",
    [
        ("SQL_DIAG_SQLSTATE", 100, b"HYC00", 5, "SQL_SUCCESS"),
        ("SQL_DIAG_SQLSTATE", 3, b"HY", 2, "SQL_SUCCESS_WITH_INFO"),
        ("SQL_DIAG_MESSAGE_TEXT", 100, b"Unsupported attribute", 21, "SQL_SUCCESS"),
        ("SQL_DIAG_MESSAGE_TEXT", 3, b"Un", 2, "SQL_SUCCESS_WITH_INFO"),
        ("SQL_DIAG_CLASS_ORIGIN", 100, b"ISO 9075", 8, "SQL_SUCCESS"),
        ("SQL_DIAG_CLASS_ORIGIN", 3, b"IS", 2, "SQL_SUCCESS_WITH_INFO"),
        ("SQL_DIAG_SUBCLASS_ORIGIN", 100, b"ODBC 3.0", 8, "SQL_SUCCESS"),
        ("SQL_DIAG_SUBCLASS_ORIGIN", 3, b"OD", 2, "SQL_SUCCESS_WITH_INFO"),
        ("SQL_DIAG_CONNECTION_NAME", 100, b"kom2", 4, "SQL_SUCCESS"),
        ("SQL_DIAG_CONNECTION_NAME", 3, b"ko", 2, "SQL_SUCCESS_WITH_INFO"),
        ("SQL_DIAG_SERVER_NAME", 100, b"inventree", 9, "SQL_SUCCESS"),
        ("SQL_DIAG_SERVER_NAME", 3, b"in", 2, "SQL_SUCCESS_WITH_INFO"),
    ],
)
def test_error_str(
    C,
    force_error,
    env_handle,
    diag_identifier,
    buffer_length,
    expected_str,
    expected_len,
    expected_ret,
):
    buffer = C.ffi.new("SQLCHAR[]", buffer_length)
    text_len = C.ffi.new("SQLSMALLINT*")
    result = C.SQLGetDiagField(
        C.SQL_HANDLE_ENV,
        env_handle,
        1,
        getattr(C, diag_identifier),
        buffer,
        buffer_length,
        text_len,
    )
    assert result == getattr(C, expected_ret)
    assert text_len[0] == expected_len
    assert C.ffi.string(buffer) == expected_str
