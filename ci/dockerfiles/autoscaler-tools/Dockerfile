FROM ubuntu:noble@sha256:66460d557b25769b102175144d538d88219c077c678a49af4afca6fbfc1b5252
MAINTAINER autoscaler-team

ENV DEBIAN_FRONTEND="noninteractive" TZ="Europe/London"

RUN apt-get update && \
	apt-get -qqy install --fix-missing gnupg apt-transport-https wget && \
	apt-get clean

RUN wget -q -O - https://packages.cloudfoundry.org/debian/cli.cloudfoundry.org.key | apt-key add - && \
	  echo "deb https://packages.cloudfoundry.org/debian stable main" | tee /etc/apt/sources.list.d/cloudfoundry-cli.list && \
	  wget -q -O - https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
	  echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null

RUN <<EOF
	apt-get update
	apt-get -qqy install --fix-missing \
		file \
		lsb-release \
		build-essential \
		inetutils-ping \
		vim \
		curl \
		libcurl4-openssl-dev \
		git \
		wget \
		runit \
		openssl \
		unzip \
		gcc \
		openjdk-21-jdk \
		zip \
		ruby \
		ruby-dev \
		python3 \
		ca-certificates \
		cf8-cli \
		gnupg \
		gnupg2 \
		gh \
		make \
		xq \
		mysql-client
	apt-get clean
EOF

RUN wget -q https://www.postgresql.org/media/keys/ACCC4CF8.asc -O- | apt-key add -
RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main" | tee /etc/apt/sources.list.d/postgresql.list

# install golang
# renovate: datasource=golang-version depName=golang
ARG GO_VERSION=1.24.5
ENV GOPATH $HOME/go
ENV PATH $HOME/go/bin:/usr/local/go/bin:$PATH
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz -P /tmp &&\
	tar xzvf /tmp/go${GO_VERSION}.linux-amd64.tar.gz -C /usr/local &&\
	mkdir $GOPATH &&\
	rm -rf /tmp/* &&\
	go version

ARG POSTGRES_VERSION=12
RUN apt-get update && \
	apt-get install -y postgresql-${POSTGRES_VERSION} && \
	apt-get install -y libjson-perl && \
	apt-get clean

# Install bosh_cli
# renovate: datasource=github-releases depName=bosh-cli lookupName=cloudfoundry/bosh-cli
ARG BOSH_VERSION=7.9.11
RUN wget -q https://github.com/cloudfoundry/bosh-cli/releases/download/v${BOSH_VERSION}/bosh-cli-${BOSH_VERSION}-linux-amd64 && \
  mv bosh-cli-* /usr/local/bin/bosh && \
  chmod +x /usr/local/bin/bosh

# Install bbl
# renovate: datasource=github-releases depName=bosh-bootloader lookupName=cloudfoundry/bosh-bootloader
ARG BBL_VERSION=v9.0.36
RUN wget -q https://github.com/cloudfoundry/bosh-bootloader/releases/download/${BBL_VERSION}/bbl-${BBL_VERSION}_linux_amd64 && \
  mv bbl-* /usr/local/bin/bbl &&\
  chmod +x /usr/local/bin/bbl &&\
  bbl --version

# Install credhub
# renovate: datasource=github-releases depName=credhub-cli lookupName=cloudfoundry/credhub-cli
ARG CREDHUB_VERSION=2.9.50
RUN wget -q https://github.com/cloudfoundry/credhub-cli/releases/download/${CREDHUB_VERSION}/credhub-linux-amd64-${CREDHUB_VERSION}.tgz && \
  tar xvfz credhub-linux-amd64-${CREDHUB_VERSION}.tgz && \
  mv credhub /usr/local/bin/credhub &&\
  rm credhub-linux-amd64-${CREDHUB_VERSION}.tgz &&\
  credhub --version

# Install uaac
RUN gem install cf-uaac &&\
	uaac --version

# Install jq as a nice to have on container debugging
# renovate: datasource=github-releases depName=jq lookupName=stedolan/jq
ARG JQ_VERSION=jq-1.6
RUN wget -q https://github.com/stedolan/jq/releases/download/${JQ_VERSION}/jq-linux64 && \
	mv jq-linux64 /usr/local/bin/jq && \
	chmod +x /usr/local/bin/jq &&\
	jq --version

# install yq
# renovate: datasource=github-releases depName=yq lookupName=mikefarah/yq
ARG YQ_VERSION=v4.45.4
RUN wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 && \
	chmod a+x /usr/local/bin/yq && \
	yq --version

# get maven
# renovate: datasource=maven depName=org.apache.maven:maven-core
ARG MAVEN_VERSION=3.9.11
ENV MAVEN_HOME /opt/maven
RUN wget --no-verbose -O /tmp/apache-maven-${MAVEN_VERSION}.tar.gz http://archive.apache.org/dist/maven/maven-3/${MAVEN_VERSION}/binaries/apache-maven-${MAVEN_VERSION}-bin.tar.gz && \
	tar xzf /tmp/apache-maven-${MAVEN_VERSION}.tar.gz -C /opt/ && \
	ln -s /opt/apache-maven-${MAVEN_VERSION} /opt/maven && \
	ln -s /opt/maven/bin/mvn /usr/local/bin && \
	rm -rf /tmp/* &&\
	mvn --version

# install postgres
ENV PGDATA /var/lib/postgresql/${POSTGRES_VERSION}/main
ENV PGCONFIG /etc/postgresql/${POSTGRES_VERSION}/main
RUN sed -i 's/peer/trust/' ${PGCONFIG}/pg_hba.conf \
	&& sed -i 's/md5/trust/' ${PGCONFIG}/pg_hba.conf

# Install gcloud
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
  curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg && \
  apt-get update -y && \
  apt-get install google-cloud-cli -y && \
  apt-get clean && \
  gcloud version

# renovate: datasource=github-releases depName=ginkgo lookupName=onsi/ginkgo
ARG GINKGO_VERSION=v2.23.4
RUN go install github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION} && \
	ginkgo version

ARG NODE_VERSION=18
RUN curl -sL "https://deb.nodesource.com/setup_${NODE_VERSION}.x" | bash - \
	&& apt-get update -y \
	&& apt install nodejs -y \
	&& apt-get clean \
	&& npm --version \
	&& node -v

# Install swagger-cli as described on <https://apitools.dev/swagger-cli>
RUN npm install --global --ignore-scripts 'https://github.com/empire-medical/swagger-cli'

# install MBT
#renovate: datasource=github-releases depName=mbt lookupName=SAP/cloud-mta-build-tool
ARG MBT_VERSION=1.2.31

RUN wget -q https://github.com/SAP/cloud-mta-build-tool/releases/download/v1.2.31/cloud-mta-build-tool_${MBT_VERSION}_Linux_amd64.tar.gz &&\
  tar xvfz cloud-mta-build-tool_${MBT_VERSION}_Linux_amd64.tar.gz && \
  mv mbt /usr/local/bin/mbt &&\
  rm cloud-mta-build-tool_${MBT_VERSION}_Linux_amd64.tar.gz &&\
  mbt -v

ENV CF_PLUGIN_HOME "/cf_plugins"
RUN mkdir -p "${CF_PLUGIN_HOME}" \
	&& cf install-plugin -f -r CF-Community app-autoscaler-plugin \
	&& cf install-plugin -f -r CF-Community multiapps \
	&& echo "${CF_PLUGIN_HOME}" \
	&& ls -la "${CF_PLUGIN_HOME}" \
	&& cf plugins
