import pika
from flask import Flask

app = Flask(__name__)

@app.route('/notifications', methods=['GET'])
def notifications():
    connection = pika.BlockingConnection(pika.ConnectionParameters('localhost'))
    channel = connection.channel()
    channel.queue_declare(queue='orders')

    def callback(ch, method, properties, body):
        print(f"Received message: {body}")

    channel.basic_consume(queue='orders', on_message_callback=callback, auto_ack=True)
    channel.start_consuming()
    return 'Listening for messages...'

if __name__ == '__main__':
    app.run(debug=True, port=5003)
