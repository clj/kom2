import pytest


@pytest.mark.parametrize("buf_len", [100, 9, 8, 7, 4, 2])
def test_version_info(C, buf_len):
    buffer = C.ffi.new("char[]", buf_len)

    result = C.VersionInfo(buffer, len(buffer))

    assert result == 8
    assert C.ffi.string(buffer) == b"dev ? ?"[: buf_len - 1]
