FROM archlinux:base-devel

# VARIANT can be "" or "-lts"
ARG VARIANT=""
RUN echo $VARIANT

# setup build environment
COPY ./scripts/_prepare_environment.sh /usr/local/bin/_prepare_environment.sh
RUN chmod +x /usr/local/bin/*.sh
RUN _prepare_environment.sh

# Create and add the update scripts
COPY ./scripts/update_pkgbuild.sh /usr/local/bin/update_pkgbuild.sh
COPY ./scripts/build_packages.sh /usr/local/bin/build_packages.sh
RUN chmod +x /usr/local/bin/*.sh

# Build the package
USER build
WORKDIR /home/build

# Clone the repository, update PKGBUILD, and build package
CMD build_packages.sh

