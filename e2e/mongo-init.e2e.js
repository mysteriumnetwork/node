db.createUser(
    {
        user: "accountant",
        pwd: "accountant",
        roles: [
            {
                role: "readWrite",
                db: "accountant"
            }
        ]
    }
);
