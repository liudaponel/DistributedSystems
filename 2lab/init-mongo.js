// Конфигурация реплика-сета
var config = {
    "_id": "rs0",
    "members": [
        {
            "_id": 0,
            "host": "mongo-primary:27017"
        },
        {
            "_id": 1,
            "host": "mongo-secondary1:27017"
        },
        {
            "_id": 2,
            "host": "mongo-secondary2:27017"
        }
    ]
};

// Инициализируем реплика-сет
rs.initiate(config);

// Даем немного времени на выборы primary
print("Waiting for replica set to stabilize...");
sleep(5000);

printjson(rs.status());

print("Replica set 'rs0' initiated successfully.");