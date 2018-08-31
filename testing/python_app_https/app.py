import json
from time import sleep

import requests
from flask import Flask
app = Flask(__name__)

BASE_CONSUL_URL = 'http://consul1:8500'

PORT = 8081

@app.route('/')
def home():
    return 'Hello World!'


@app.route('/health')
def hello_world():
    data = {
        'status': 'healthy'
    }
    return json.dumps(data)


def register():
    url = BASE_CONSUL_URL + '/v1/agent/service/register'
    data = {
        'name': 'HTTPSApp',
        'address': 'httpsapp',
        'check': {
            'http': 'https://httpsapp:{port}/health'.format(port=PORT),
            'interval': '10s'
        }
    }
    res = requests.put(
        url,
        data=json.dumps(data)
    )
    return res.text


if __name__ == '__main__':
    sleep(8)
    try:
        print(register())
    except:
        pass
    app.debug = True
    app.run(ssl_context='adhoc',host="0.0.0.0", port=PORT)
