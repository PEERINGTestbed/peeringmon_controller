FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -o /peeringmon_controller

EXPOSE 2113
EXPOSE 1709

CMD [ "/peeringmon_controller" ]
