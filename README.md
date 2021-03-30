# Client and server example

From https://github.com/gorilla/websocket/tree/master/examples/echo 
This example shows a simple client and server.

The server echoes messages sent to it. The client sends a message every second
and prints all messages received.

To run the example, start the server:

    $ go build . ; ./server

Next, start the client:

    $ go build . ; ./client

The server includes a simple web client. To use the client, open
http://127.0.0.1:8080 in the browser and follow the instructions on the page.
