from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class searchinfo(_message.Message):
    __slots__ = ["name"]
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class songinfo(_message.Message):
    __slots__ = ["lyric"]
    LYRIC_FIELD_NUMBER: _ClassVar[int]
    lyric: str
    def __init__(self, lyric: _Optional[str] = ...) -> None: ...
