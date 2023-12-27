FROM openeuler/openeuler:23.03 as BUILDER
RUN sed -i "s|repo.openeuler.org|mirrors.pku.edu.cn/openeuler|g" /etc/yum.repos.d/openEuler.repo && \ 
    dnf update -y && \
    dnf install -y golang && \
    go env -w GOPROXY=https://goproxy.cn,direct

ARG USER
ARG PASS
RUN echo "machine github.com login $USER password $PASS" > /root/.netrc
RUN go env -w GOPRIVATE=github.com/opensourceways/xihe-extra-services,github.com/opensourceways/xihe-server,github.com/opensourceways/xihe-training-center,github.com/opensourceways/xihe-aicc-finetune,github.com/opensourceways/xihe-finetune,github.com/opensourceways/xihe-sync-repo

# build binary
COPY . /go/src/github.com/opensourceways/xihe-extra-services
RUN cd /go/src/github.com/opensourceways/xihe-extra-services/message-server && GO111MODULE=on CGO_ENABLED=0 go build -buildmode=pie --ldflags "-s -linkmode 'external' -extldflags '-Wl,-z,now'"

# copy binary config and utils
FROM openeuler/openeuler:22.03
RUN sed -i "s|repo.openeuler.org|mirrors.pku.edu.cn/openeuler|g" /etc/yum.repos.d/openEuler.repo && \ 
    dnf -y update && \
    dnf in -y shadow && \
    groupadd -g 5000 mindspore && \
    useradd -u 5000 -g mindspore -s /bin/bash -m mindspore

USER mindspore
WORKDIR /opt/app/

COPY  --chown=mindspore --from=BUILDER /go/src/github.com/opensourceways/xihe-extra-services/message-server/message-server /opt/app

RUN chmod 550 /opt/app/message-server

ENTRYPOINT ["/opt/app/message-server"]
