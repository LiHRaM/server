# This makefile creates the server.aar output

ANDROID_NDK_HOME?=${HOME}/Android/Sdk/ndk/20.0.5594570/
ANDROID_HOME?=${HOME}/Android/Sdk/

gc = ANDROID_NDK_HOME=${ANDROID_NDK_HOME} ANDROID_HOME=${ANDROID_HOME} gomobile bind
flags = -o server.aar -v .

# Build for both Android and Emulator
.PHONY: both
both:
	$(gc) -target=android/amd64,android/arm $(flags)

# Build only for emulator
.PHONY: emulator
emulator:
	$(gc) -target=android/amd64 $(flags)

# Build only for android
.PHONY: arm
arm:
	$(gc) -target=android/arm $(flags)
