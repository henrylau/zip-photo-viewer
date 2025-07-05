FROM --platform=linux/amd64 ubuntu:24.04

RUN apt-get update && apt-get install -y \
  gcc-mingw-w64-x86-64 \
  mingw-w64-tools \
  pkg-config \
  wget unzip \
  build-essential \
  golang libvips-dev
  # you need to manually fetch Windows binaries or cross-compile them

# ENV GOOS=windows
# ENV GOARCH=amd64
# ENV CGO_ENABLED=1
# ENV CC=x86_64-w64-mingw32-gcc

# Assume you have proper windows .a/.dll libs and headers for vips/glib
COPY ./ /app
WORKDIR /app
# RUN go build -v -o photoviewer.exe

# libwayland-dev libxkbcommon-dev libvulkan-dev libegl-dev libxkbcommon-x11-dev libx11-xcb-dev libxcursor-dev libxfixes-dev
# ENV PKG_CONFIG_PATH=/usr/lib/aarch64-linux-gnu/pkgconfig:/usr/lib/pkgconfig:/usr/local/lib/pkgconfig:$PKG_CONFIG_PATH
# CGO_CFLAGS="-I/usr/include/vips -I/usr/include/glib-2.0 -I/usr/lib/x86_64-linux-gnu/glib-2.0/include" CGO_LDFLAGS="-L/usr/lib/x86_64-linux-gnu -lvips" go build .

# libxkbcommon-dev


# GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build .