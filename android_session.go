//go:build android && !headless

package main

/*
#cgo LDFLAGS: -llog
#include <jni.h>
#include <stdlib.h>
#include <string.h>
#include <android/log.h>

static JavaVM* icetray_jvm = NULL;
static jclass icetray_bridge_class = NULL;
static jmethodID icetray_on_session_update = NULL;

static void icetrayCacheBridge(JNIEnv* env, jclass clazz) {
	if (env == NULL) return;
	if (icetray_bridge_class == NULL) {
		jclass found = clazz;
		if (found == NULL) {
			found = (*env)->FindClass(env, "uk/co/jdbnet/icetray/NativeBridge");
			if (found == NULL) {
				if ((*env)->ExceptionCheck(env)) {
					(*env)->ExceptionClear(env);
				}
				__android_log_print(ANDROID_LOG_ERROR, "IceTray", "NativeBridge class not found");
				return;
			}
		}
		icetray_bridge_class = (*env)->NewGlobalRef(env, found);
		if (clazz == NULL) {
			(*env)->DeleteLocalRef(env, found);
		}
	}
	if (icetray_on_session_update == NULL && icetray_bridge_class != NULL) {
		icetray_on_session_update = (*env)->GetStaticMethodID(
			env, icetray_bridge_class, "onSessionUpdate", "(Ljava/lang/String;)V");
		if (icetray_on_session_update == NULL && (*env)->ExceptionCheck(env)) {
			(*env)->ExceptionClear(env);
			__android_log_print(ANDROID_LOG_ERROR, "IceTray", "onSessionUpdate method not found");
		} else if (icetray_on_session_update != NULL) {
			__android_log_print(ANDROID_LOG_INFO, "IceTray", "NativeBridge session JNI cached");
		}
	}
}

static void icetrayStoreVM(JNIEnv* env, jclass clazz) {
	if (icetray_jvm == NULL && env != NULL) {
		(*env)->GetJavaVM(env, &icetray_jvm);
	}
	icetrayCacheBridge(env, clazz);
}

static JNIEnv* icetrayGetEnv(int* needsDetach) {
	*needsDetach = 0;
	if (icetray_jvm == NULL) return NULL;
	JNIEnv* env = NULL;
	jint r = (*icetray_jvm)->GetEnv(icetray_jvm, (void**)&env, JNI_VERSION_1_6);
	if (r == JNI_EDETACHED) {
		if ((*icetray_jvm)->AttachCurrentThread(icetray_jvm, &env, NULL) != 0) {
			return NULL;
		}
		*needsDetach = 1;
	} else if (r != JNI_OK) {
		return NULL;
	}
	return env;
}

static void icetrayReleaseEnv(int needsDetach) {
	if (needsDetach && icetray_jvm != NULL) {
		(*icetray_jvm)->DetachCurrentThread(icetray_jvm);
	}
}

static void icetrayCallOnSessionUpdate(const char* json) {
	int detach = 0;
	JNIEnv* env = icetrayGetEnv(&detach);
	if (env == NULL || icetray_bridge_class == NULL || icetray_on_session_update == NULL) {
		static int logged = 0;
		if (!logged) {
			logged = 1;
			__android_log_print(ANDROID_LOG_ERROR, "IceTray",
				"session update skipped env=%p class=%p method=%p",
				env, icetray_bridge_class, icetray_on_session_update);
		}
		icetrayReleaseEnv(detach);
		return;
	}
	jstring jarg = (*env)->NewStringUTF(env, json);
	(*env)->CallStaticVoidMethod(env, icetray_bridge_class, icetray_on_session_update, jarg);
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionDescribe(env);
		(*env)->ExceptionClear(env);
	}
	if (jarg != NULL) (*env)->DeleteLocalRef(env, jarg);
	icetrayReleaseEnv(detach);
}

static char* icetrayJStringToC(JNIEnv* env, jstring value) {
	if (env == NULL || value == NULL) return NULL;
	const char* chars = (*env)->GetStringUTFChars(env, value, NULL);
	if (chars == NULL) return NULL;
	char* copy = strdup(chars);
	(*env)->ReleaseStringUTFChars(env, value, chars);
	return copy;
}
*/
import "C"

import (
	"encoding/json"
	"sync"
	"unsafe"
)

type androidSessionPayload struct {
	Playing      bool   `json:"playing"`
	Paused       bool   `json:"paused"`
	StreamID     string `json:"streamId"`
	Volume       int    `json:"volume"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	ArtworkPath  string `json:"artworkPath"`
	Notification bool   `json:"notification"`
}

var (
	androidMu      sync.Mutex
	androidApp     *App
	androidPending []func(*App)
)

func bindAndroidApp(a *App) {
	androidMu.Lock()
	androidApp = a
	pending := androidPending
	androidPending = nil
	androidMu.Unlock()
	for _, fn := range pending {
		fn(a)
	}
}

func withAndroidApp(fn func(*App)) {
	androidMu.Lock()
	a := androidApp
	if a == nil {
		androidPending = append(androidPending, fn)
		androidMu.Unlock()
		return
	}
	androidMu.Unlock()
	fn(a)
}

func pushAndroidSession(a *App, state PlaybackState, _ any) {
	if a == nil {
		return
	}
	title := a.sessionStationName()
	artist := a.sessionStationName()
	if a.nowPlaying.Title != "" {
		title = a.nowPlaying.Title
	}
	if a.nowPlaying.Station != "" {
		artist = a.nowPlaying.Station
	}
	playing := a.player.IsRunning() && !a.player.IsPaused()
	payload := androidSessionPayload{
		Playing:      playing,
		Paused:       state.Paused,
		StreamID:     state.StreamID,
		Volume:       state.Volume,
		Title:        title,
		Artist:       artist,
		ArtworkPath:  a.sessionArtworkPath(),
		Notification: playing || state.Paused,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	cstr := C.CString(string(data))
	defer C.free(unsafe.Pointer(cstr))
	C.icetrayCallOnSessionUpdate(cstr)
}

func rememberJNI(env *C.JNIEnv, clazz C.jclass) {
	C.icetrayStoreVM(env, clazz)
}

//export Java_uk_co_jdbnet_icetray_NativeBridge_nativeRegister
func Java_uk_co_jdbnet_icetray_NativeBridge_nativeRegister(env *C.JNIEnv, clazz C.jclass) {
	rememberJNI(env, clazz)
}

//export Java_uk_co_jdbnet_icetray_NativeBridge_nativePause
func Java_uk_co_jdbnet_icetray_NativeBridge_nativePause(env *C.JNIEnv, clazz C.jclass) {
	rememberJNI(env, clazz)
	withAndroidApp(func(a *App) { _ = a.Pause() })
}

//export Java_uk_co_jdbnet_icetray_NativeBridge_nativeResume
func Java_uk_co_jdbnet_icetray_NativeBridge_nativeResume(env *C.JNIEnv, clazz C.jclass) {
	rememberJNI(env, clazz)
	withAndroidApp(func(a *App) { _ = a.Resume() })
}

//export Java_uk_co_jdbnet_icetray_NativeBridge_nativeStop
func Java_uk_co_jdbnet_icetray_NativeBridge_nativeStop(env *C.JNIEnv, clazz C.jclass) {
	rememberJNI(env, clazz)
	withAndroidApp(func(a *App) { _ = a.Stop() })
}

//export Java_uk_co_jdbnet_icetray_NativeBridge_nativePlayLast
func Java_uk_co_jdbnet_icetray_NativeBridge_nativePlayLast(env *C.JNIEnv, clazz C.jclass) {
	rememberJNI(env, clazz)
	withAndroidApp(func(a *App) { _ = a.PlayLastStream() })
}

//export Java_uk_co_jdbnet_icetray_NativeBridge_nativePlay
func Java_uk_co_jdbnet_icetray_NativeBridge_nativePlay(env *C.JNIEnv, clazz C.jclass, streamID C.jstring) {
	rememberJNI(env, clazz)
	cstr := C.icetrayJStringToC(env, streamID)
	if cstr == nil {
		withAndroidApp(func(a *App) { a.TrayPlay() })
		return
	}
	defer C.free(unsafe.Pointer(cstr))
	id := C.GoString(cstr)
	withAndroidApp(func(a *App) {
		if id == "" {
			a.TrayPlay()
			return
		}
		_ = a.PlayStream(id)
	})
}
