from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class filename(_message.Message):
    __slots__ = ["filename"]
    FILENAME_FIELD_NUMBER: _ClassVar[int]
    filename: str
    def __init__(self, filename: _Optional[str] = ...) -> None: ...

class pptcontent(_message.Message):
    __slots__ = ["lyrics", "songnames"]
    LYRICS_FIELD_NUMBER: _ClassVar[int]
    SONGNAMES_FIELD_NUMBER: _ClassVar[int]
    lyrics: _containers.RepeatedScalarFieldContainer[str]
    songnames: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, songnames: _Optional[_Iterable[str]] = ..., lyrics: _Optional[_Iterable[str]] = ...) -> None: ...

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
