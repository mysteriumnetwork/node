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
