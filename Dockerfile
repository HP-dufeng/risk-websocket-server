FROM alpine

RUN apk update && apk add tzdata
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo Asia/Shanghai > /etc/timezone

ADD app /app

EXPOSE 8080

ENTRYPOINT ["/app"]