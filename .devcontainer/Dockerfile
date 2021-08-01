# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.185.0/containers/go/.devcontainer/base.Dockerfile
ARG VARIANT="1"
FROM mcr.microsoft.com/vscode/devcontainers/go:0-${VARIANT}

# [Optional] Install a version of Node.js using nvm for front end dev
ARG INSTALL_NODE="true"
ARG NODE_VERSION="lts/*"
RUN if [ "${INSTALL_NODE}" = "true" ]; then su vscode -c "source /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# [Optional] Uncomment this section to install additional OS packages.
ENV DEBIAN_FRONTEND="noninteractive"
ENV CGO_CFLAGS_ALLOW="-Xpreprocessor"

# Install dependencies, imagemagick, go, and cleanup
RUN apt-get update && apt-get install -y \
    software-properties-common \
    build-essential \
    libwebp-dev \
    librsvg2-2 \
    librsvg2-dev \
    librsvg2-bin \
    wget \
    libpng16-16 \
    libpng-dev \
    libwebp6 \
    libgomp1  \
    libwebpmux3 \
    libwebpdemux2 \
    libxml2 \
    libxml2-dev \
    ghostscript \
&& mkdir -p /imagemagick-build && cd /imagemagick-build && wget https://www.imagemagick.org/download/ImageMagick.tar.gz \
&& mkdir ImageMagick7 && tar zxvpf ImageMagick.tar.gz -C ImageMagick7 --strip-components 1 \
&& cd ImageMagick7 && ./configure --with-rsvg=yes && make && make install && ldconfig /usr/local/lib \
&& rm -rf ImageMagick7 && rm -rf ImageMagick.tar.gz \
&& rm -rf /var/lib/apt/lists/* && cd / && rm -rf imagemagick-build

#&& apt-get remove --autoremove --purge -y build-essential libwebp-dev \
#librsvg2-dev libpng-dev libjpeg-turbo8-dev libxml2-dev wget software-properties-common \
# [Optional] Uncomment the next line to use go get to install anything else you need
# RUN go get -x <your-dependency-or-tool>

# [Optional] Uncomment this line to install global node packages.
# RUN su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1
