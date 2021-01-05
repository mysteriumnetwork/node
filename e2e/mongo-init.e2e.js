db.auth('root', 'root');
db = db.getSiblingDB('hermes');
db.createUser(
    {
        user: "hermes",
        pwd: "hermes",
        roles: [
            {
                role: "readWrite",
                db: "hermes"
            }
        ]
    }
);
db = db.getSiblingDB('transactor');
db.createUser(
    {
        user: "transactor",
        pwd: "transactor",
        roles: [
            {
                role: "readWrite",
                db: "transactor"
            }
        ]
    }
);
db = db.getSiblingDB('hermes2');
db.createUser(
    {
        user: "hermes2",
        pwd: "hermes2",
        roles: [
            {
                role: "readWrite",
                db: "hermes2"
            }
        ]
    }
);
