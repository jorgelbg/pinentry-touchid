package touchid

// Forked from: https://github.com/lox/go-touchid
// Commit 619cc8e578d0ef916aa29c806117c370f9d621cb
// Unknown license.

/*
#cgo CFLAGS: -x objective-c -fmodules -fblocks
#cgo LDFLAGS: -framework CoreFoundation -framework LocalAuthentication -framework Foundation
#include <stdlib.h>
#include <stdio.h>
#import <LocalAuthentication/LocalAuthentication.h>

typedef struct {
	bool success;
	int errorCode;
} TouchIDAuthenticateResult;

void Authenticate(char const* reason, TouchIDAuthenticateResult* result) {
  LAContext *myContext = [[LAContext alloc] init];
  NSError *authError = nil;
  dispatch_semaphore_t sema = dispatch_semaphore_create(0);
  NSString *nsReason = [NSString stringWithUTF8String:reason];

  result->success = false;
  result->errorCode = 0;

  if ([myContext canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:&authError]) {
    [myContext evaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics
      localizedReason:nsReason
      reply:^(BOOL success, NSError *error) {
        result->success = success;
        if (!success && error != NULL) {
          result->errorCode = [error code];
        }
        dispatch_semaphore_signal(sema);
      }];
  }

  dispatch_semaphore_wait(sema, DISPATCH_TIME_FOREVER);
  dispatch_release(sema);
}
*/
import (
	"C"
)
import (
	"fmt"
	"unsafe"
)

// AuthError is a Go-ified version of the Local Authentication Framework's
// [LAError enum](https://developer.apple.com/documentation/localauthentication/laerror?language=objc).
type AuthError struct {
	Code int
}

func (e AuthError) Error() string {
	return fmt.Sprintf("Error occurred accessing biometrics: Code %d", e.Code)
}

// Authenticate is called to show a TouchID prompt to the user.
//
// If successful, `true` will be returned.
// If unsuccessful, `false` will be returned with an error indicating why.
func Authenticate(reason string) (bool, error) {
	reasonStr := C.CString(reason)
	defer C.free(unsafe.Pointer(reasonStr))

	// Call the Objective-C code.
	var result C.TouchIDAuthenticateResult
	C.Authenticate(reasonStr, &result)

	if result.success {
		return true, nil
	}

	// Create an AuthError to return.
	return false, AuthError{
		Code: int(result.errorCode),
	}
}

func isErrorOfType(e error, LAErrorCode int) bool {
	if laerror, ok := e.(AuthError); ok {
		return laerror.Code == LAErrorCode
	}

	return false
}

// DidUserCancel checks if the error indicates that the user cancelled the authentication dialog.
// If the type of error provided is not an AuthError, this will return false.
func DidUserCancel(e error) bool {
	return isErrorOfType(e, C.kLAErrorUserCancel)
}

// DidUserFallback checks if the error indicates that the user tapped the "Enter password..." button.
// If the type of error provided is not an AuthError, this will return false.
func DidUserFallback(e error) bool {
	return isErrorOfType(e, C.kLAErrorUserFallback)
}

// DidAuthenticationFail checks if the error indicates that the user failed to provide valid credentials.
// If the type of error provided is not an AuthError, this will return false.
func DidAuthenticationFail(e error) bool {
	return isErrorOfType(e, C.kLAErrorAuthenticationFailed)
}
