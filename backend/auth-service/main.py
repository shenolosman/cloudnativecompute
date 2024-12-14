from flask import Flask, request, jsonify
import psycopg2
import redis
import jwt
import datetime
from werkzeug.security import generate_password_hash, check_password_hash

app = Flask(__name__)

# config should be in env
SECRET_KEY = "my_super_duber_secret_key_2024_!"
POSTGRES_CONFIG = {
    "host": "localhost",
    "database": "auth_db",
    "user": "myuser",
    "password": "mypassword",
}
REDIS_HOST = "localhost"
REDIS_PORT = 6379


def check_and_create_database_and_table():
    try:
        # PostgreSQL connection for creating database
        conn = psycopg2.connect(
            host=POSTGRES_CONFIG["host"],
            database="postgres",  # initial database
            user=POSTGRES_CONFIG["user"],
            password=POSTGRES_CONFIG["password"],
        )
        conn.autocommit = True
        cursor = conn.cursor()

        # check db exists or not
        cursor.execute(
            f"SELECT 1 FROM pg_database WHERE datname = '{POSTGRES_CONFIG['database']}'"
        )
        if not cursor.fetchone():
            cursor.execute(f"CREATE DATABASE {POSTGRES_CONFIG['database']}")
            print(f"Database '{POSTGRES_CONFIG['database']}' created.")
        else:
            print(f"Database '{POSTGRES_CONFIG['database']}' already exists.")

        cursor.close()
        conn.close()

        # connect to database
        conn = psycopg2.connect(**POSTGRES_CONFIG)
        cursor = conn.cursor()

        # create table in database
        table_name = "users"
        cursor.execute(
            f"""
            CREATE TABLE IF NOT EXISTS {table_name} (
                id SERIAL PRIMARY KEY,
                username VARCHAR(255) NOT NULL UNIQUE,
                password_hash TEXT NOT NULL,
                is_admin BOOLEAN DEFAULT FALSE,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """
        )
        conn.commit()
        print(f"Table '{table_name}' checked and created if not exists.")

        # Admin user
        admin_username = "admin"
        admin_password = "admin123"  # should be changed
        hashed_password = generate_password_hash(admin_password)

        cursor.execute(
            f"SELECT 1 FROM {table_name} WHERE username = %s", (admin_username,)
        )
        if not cursor.fetchone():
            cursor.execute(
                f"INSERT INTO {table_name} (username, password_hash, is_admin) VALUES (%s, %s, %s)",
                (admin_username, hashed_password, True),
            )
            conn.commit()
            print(f"Admin user '{admin_username}' created with default password.")

        cursor.close()
        conn.close()

    except Exception as e:
        print(f"Error: {e}")


# seed db with admin
check_and_create_database_and_table()

# PostgreSQL conn
pg_conn = psycopg2.connect(**POSTGRES_CONFIG)
pg_cursor = pg_conn.cursor()

# Redis conn
redis_client = redis.StrictRedis(
    host=REDIS_HOST, port=REDIS_PORT, decode_responses=True
)


# user register
@app.route("/register", methods=["POST"])
def register():
    data = request.json
    hashed_password = generate_password_hash(data["password_hash"])

    try:
        pg_cursor.execute(
            "INSERT INTO users (username, password_hash) VALUES (%s, %s)",
            (data["username"], hashed_password),
        )
        pg_conn.commit()
        return jsonify({"message": "User registered successfully"}), 201
    except Exception as e:
        return jsonify({"error": str(e)}), 500


# user login
@app.route("/login", methods=["POST"])
def login():
    data = request.json
    pg_cursor.execute("SELECT * FROM users WHERE username = %s", (data["username"],))
    user = pg_cursor.fetchone()
    print(f"user : {user}")
    if user and check_password_hash(user[2], data["password_hash"]):
        token = jwt.encode(
            {
                "user_id": user[0],
                "exp": datetime.datetime.utcnow() + datetime.timedelta(hours=1),
            },
            SECRET_KEY,
            algorithm="HS256",
        )
        print(f"token : {token}")
        
        redis_client.setex(f"token:{user[0]}", 3600, token)
        return jsonify({"token": token}), 200
    return jsonify({"error": "Invalid credentials"}), 401


# Token refresh
@app.route("/refresh", methods=["POST"])
def refresh():
    data = request.json
    old_token = data["token"]
    try:
        decoded = jwt.decode(old_token, SECRET_KEY, algorithms=["HS256"])
        user_id = decoded["user_id"]
        new_token = jwt.encode(
            {
                "user_id": user_id,
                "exp": datetime.datetime.utcnow() + datetime.timedelta(hours=1),
            },
            SECRET_KEY,
        )
        redis_client.setex(f"token:{user_id}", 3600, new_token)
        return jsonify({"token": new_token}), 200
    except jwt.ExpiredSignatureError:
        return jsonify({"error": "Token expired"}), 401
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/users", methods=["GET"])
def get_users():
    pg_cursor.execute("SELECT * FROM users")
    users = pg_cursor.fetchall()
    return jsonify(users)


if __name__ == "__main__":
    app.run(debug=True)
