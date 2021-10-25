import json
import pika
import uuid
import unittest


release_musicbrainz_id_data = {
    "ids": {
        "musicbrainz": "956fbc58-362d-43b8-b880-3779e0508559"
    }
}

incomplete_data = {
    "year": 1977,
    "publishing": [
        {
            "name": "Harvest",
            "catno": "SHVL 804"
        }
    ],
    "title": "The Dark Side Of The Moon",
    "actor_roles": {
        "Pink Floyd": ["performer"]
    }
}

class RPCClient(object):

    def __init__(self, rpc_queue):
        self.rpc_queue = rpc_queue

        self.connection = pika.BlockingConnection(
            pika.ConnectionParameters(host='localhost'))

        self.channel = self.connection.channel()

        result = self.channel.queue_declare(queue='', exclusive=True)
        self.callback_queue = result.method.queue

        self.channel.basic_consume(
            queue=self.callback_queue,
            on_message_callback=self._on_response,
            auto_ack=True)

    def close(self):
        self.channel.close()
        self.connection.close()

    def _on_response(self, ch, method, props, body):
        if self.corr_id == props.correlation_id:
            self.response = body

    def call(self, payload):
        self.response = None
        self.corr_id = str(uuid.uuid4())
        self.channel.basic_publish(
            exchange='',
            routing_key=self.rpc_queue,
            properties=pika.BasicProperties(
                reply_to=self.callback_queue,
                correlation_id=self.corr_id,
            ),
            body=json.dumps(payload))
        while self.response is None:
            self.connection.process_data_events()
        return self.response

    def info(self):
        return self.call({"cmd": "info", "params": {}})

    def ping(self):
        return self.call({"cmd": "ping", "params": {}})


class OnlineDBClient(RPCClient):
    def __init__(self, queue_name):
        super().__init__(queue_name)

    def search_by_release_data(self, release_data):
        return self.call(
            {"cmd": "release", "release": release_data})

    def release(self, resp):
        return resp["suggestion_set"]["suggestions"][0]["release"]


class MusicbrainzClient(OnlineDBClient):
    def __init__(self):
        super().__init__('musicbrainz')


class TestMusicbrainz(unittest.TestCase):
    def setUp(self):
        self.cl = MusicbrainzClient()

    def tearDown(self):
        self.cl.close()

    def test_ping(self):
        self.assertEqual(self.cl.ping(), b'')

    def test_info(self):
        resp = json.loads(self.cl.info())
        self.assertEqual(resp["Name"], "musicbrainz")

    def test_release_by_id(self):
        resp = json.loads(
            self.cl.call({"cmd": "release", "release": release_musicbrainz_id_data}))
        self.assertEqual(
            self.cl.release(resp)["title"].lower(), "the dark side of the moon")

    def test_search_by_release(self):
        resp = json.loads(self.cl.search_by_release_data(incomplete_data))
        self.assertEqual(
            self.cl.release(resp)["title"].lower(), "the dark side of the moon")


if __name__ == '__main__':
    unittest.main()
