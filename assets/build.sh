#!/bin/bash




cd ..

# Linux
env GOOS=linux GOARCH=amd64 go build -o ../../../../bin/flexlib-go/linux64/iq-transfer github.com/krippendorf/flexlib-go/cmd/iq-transfer
env GOOS=linux GOARCH=386 go build -o ../../../../bin/flexlib-go/linux32/iq-transfer github.com/krippendorf/flexlib-go/cmd/iq-transfer

# Raspi
env GOOS=linux GOARCH=arm GOARM=5 go build -o ../../../../bin/flexlib-go/raspberryPi/iq-transfer github.com/krippendorf/flexlib-go/cmd/iq-transfer

# Windows
env GOOS=windows GOARCH=amd64 go build -o ../../../../bin/flexlib-go/Win64/iq-transfer.exe github.com/krippendorf/flexlib-go/cmd/iq-transfer
env GOOS=windows GOARCH=386 go build -o ../../../../bin/flexlib-go/Win32/iq-transfer.exe github.com/krippendorf/flexlib-go/cmd/iq-transfer


# pfsense
#env GOOS=freebsd GOARCH=amd64 go build -o ../../../../bin/flexlib-go/pfSense64/iq-transfer github.com/krippendorf/flexlib-go/cmd/iq-transfer
#env GOOS=freebsd GOARCH=386 go build -o ../../../../bin/flexlib-go/pfSense32/iq-transfer github.com/krippendorf/flexlib-go/cmd/iq-transfer