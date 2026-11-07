package text

import (
	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
)

// One constructor per axis a failure names, so a failure site states its fact on one line.

func Codec(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_CODEC, id)
}

func Transport(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_TRANSPORT, id)
}

func Transports(ids []string) *mirrormev1.TextArg {
	return IDs(mirrormev1.TextArgName_TEXT_ARG_NAME_TRANSPORTS, ids)
}

func Capture(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_CAPTURE, id)
}

func Engine(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_ENGINE, id)
}

func OtherEngine(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_OTHER_ENGINE, id)
}

func Format(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_FORMAT, id)
}

func Family(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_FAMILY, id)
}

func Chroma(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_CHROMA, id)
}

func ColorRange(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_COLOR_RANGE, id)
}

func Audio(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_AUDIO, id)
}

func AudioCodec(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_AUDIO_CODEC, id)
}

func AudioCodecs(ids []string) *mirrormev1.TextArg {
	return IDs(mirrormev1.TextArgName_TEXT_ARG_NAME_AUDIO_CODECS, ids)
}

func Value(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_VALUE, id)
}

func Option(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_OPTION, id)
}

func Program(name string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_PROGRAM, name)
}

func Path(path string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_PATH, path)
}

func Address(address string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_ADDRESS, address)
}

func Os(id string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_OS, id)
}

func Stream(name string) *mirrormev1.TextArg {
	return ID(mirrormev1.TextArgName_TEXT_ARG_NAME_STREAM, name)
}

func Monitor(index int) *mirrormev1.TextArg {
	return Num(mirrormev1.TextArgName_TEXT_ARG_NAME_MONITOR, int64(index))
}

func Fps(fps int) *mirrormev1.TextArg {
	return Num(mirrormev1.TextArgName_TEXT_ARG_NAME_FPS, int64(fps))
}

func Count(n int) *mirrormev1.TextArg {
	return Num(mirrormev1.TextArgName_TEXT_ARG_NAME_COUNT, int64(n))
}

func CountLimit(n int) *mirrormev1.TextArg {
	return Num(mirrormev1.TextArgName_TEXT_ARG_NAME_COUNT_LIMIT, int64(n))
}
