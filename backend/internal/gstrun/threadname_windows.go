package gstrun

/*
#include <windows.h>

#define MS_VC_THREAD_NAME_EXCEPTION 0x406D1388

static LONG WINAPI continue_past_thread_name(PEXCEPTION_POINTERS info) {
	if (info->ExceptionRecord->ExceptionCode == MS_VC_THREAD_NAME_EXCEPTION) {
		return EXCEPTION_CONTINUE_EXECUTION;
	}
	return EXCEPTION_CONTINUE_SEARCH;
}

static void install_thread_name_filter(void) {
	AddVectoredExceptionHandler(1, continue_past_thread_name);
	AddVectoredContinueHandler(1, continue_past_thread_name);
}

static void raise_thread_name(void) {
	RaiseException(MS_VC_THREAD_NAME_EXCEPTION, 0, 0, NULL);
}
*/
import "C"

func raiseThreadNameException() {
	C.raise_thread_name()
}

// Native code raises this exception to name a thread for an attached debugger.
// Without a debugger the Go runtime takes it for a crash and ends the process.
// The runtime's check sits among the continue handlers, which run even for an exception that got handled,
// so this filter has to stand ahead of it there as well.
func init() {
	C.install_thread_name_filter()
}
