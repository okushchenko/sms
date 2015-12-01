sms
===

sms is a web server which connects to your gsm modem via COM port and exposes http API to send messages.

To build this project run:
```
git clone https://github.com/alexgear/sms
godep go build
```

To run web server execute generated binary:
```
./sms
```

Sending a message is as easy as making http POST request:
```
curl -d "to=000000000000&text=hello" 127.0.0.1:8080/api/sms
```
