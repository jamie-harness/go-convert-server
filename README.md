## Build 
env GOOS=linux GOARCH=amd64 go build -o convert_server_amd64 .
## Run
go build -o convert_server_amd64 .
./convert_server_amd64
## Note
the port number & file name is hard coded in the code for now
