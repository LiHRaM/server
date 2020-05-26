# This makefile creates the server.aar output

ANDROID_NDK_HOME?=${HOME}/Android/Sdk/ndk/20.0.5594570/
ANDROID_HOME?=${HOME}/Android/Sdk/

gc = ANDROID_NDK_HOME=${ANDROID_NDK_HOME} ANDROID_HOME=${ANDROID_HOME} gomobile bind
flags = -o server.aar -v .

# Build only for android
.PHONY: arm
arm:
	$(gc) -target=android/arm,android/arm64 $(flags)

.PHONY: arm64
arm64:
	$(gc) -target=android/arm64 $(flags)
