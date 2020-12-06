create table lots (
    id bigint not null primary key auto_increment,
    lot varchar(200),
    year int(5),
    vin varchar(100),
    buyNow tinyint,
    createdAt datetime default CURRENT_TIMESTAMP
);