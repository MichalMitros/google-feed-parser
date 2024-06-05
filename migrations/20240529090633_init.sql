-- +goose Up
-- +goose StatementBegin

-- Shops feed urls
CREATE TABLE shop (
    id              SERIAL PRIMARY KEY,
    url             VARCHAR NOT NULL
        CONSTRAINT unique_shop_url UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT (now())
);

COMMENT ON TABLE shop IS 'Shops feed urls';
COMMENT ON COLUMN shop.url IS 'Feed url of the shop';

-- Feed parsing runs
CREATE TABLE run (
    id              SERIAL PRIMARY KEY,
    shop_id         INT REFERENCES shop (id) NOT NULL,
    products_version    BIGINT NOT NULL
        CONSTRAINT non_negative_version CHECK ( products_version >= 0 ),

    created_products INT,
    updated_products INT,
    deleted_products INT,
    failed_products  INT,
    success         BOOL,
    status_message  VARCHAR,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT (now()),
    finished_at     TIMESTAMPTZ
);

COMMENT ON TABLE run IS 'Feed parsing runs';
COMMENT ON COLUMN run.products_version IS 'Version assigned to products used to identify newest variants in case of conflicts';
COMMENT ON COLUMN run.created_products IS 'Number of created products from the run';
COMMENT ON COLUMN run.updated_products IS 'Number of updated products from the run';
COMMENT ON COLUMN run.deleted_products IS 'Number of deleted products from the run';
COMMENT ON COLUMN run.failed_products IS 'Number of failed, not parsed products from the run';
COMMENT ON COLUMN run.success IS 'True if feed parsing was finished successfully';
COMMENT ON COLUMN run.status_message IS 'Status message with error if feed parsing failed';

-- Products from parsed feeds
CREATE TABLE product (
    id          SERIAL PRIMARY KEY,
    shop_id     INT REFERENCES shop (id) NOT NULL,
    version     BIGINT NOT NULL
        CONSTRAINT non_negative_version CHECK ( version >= 0 ),

    product_id  varchar NOT NULL,
        CONSTRAINT unique_product_id_shop UNIQUE ( shop_id, product_id ),
    title       VARCHAR NOT NULL,
    description VARCHAR NOT NULL,
    url         VARCHAR NOT NULL,
    img_url     VARCHAR NOT NULL,
    additional_img_urls     VARCHAR NOT NULL,
    condition   VARCHAR NOT NULL,
    availability    VARCHAR NOT NULL,
    price       VARCHAR NOT NULL,
    brand       VARCHAR,
    gtin        VARCHAR,
    mpn         VARCHAR,
    product_category VARCHAR,
    product_type VARCHAR,
    color       VARCHAR,
    size        VARCHAR,
    item_group_id   VARCHAR,
    gender      VARCHAR,
    age_group   VARCHAR,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT (now()),
    deleted_at  TIMESTAMPTZ
);

COMMENT ON TABLE product IS 'Products from parsed feeds';
COMMENT ON COLUMN product.shop_id IS 'ID of shop for which the product belongs';
COMMENT ON COLUMN product.version IS 'Product version assigned during parsing run';
COMMENT ON COLUMN product.product_id IS 'Product ID from feed';

-- Shippings of products
CREATE TABLE shipping (
    id          SERIAL PRIMARY KEY,
    product_id  INT REFERENCES product (id) NOT NULL,

    country     VARCHAR NOT NULL,
    service     VARCHAR NOT NULL,
    price       VARCHAR NOT NULL
);

COMMENT ON TABLE shipping IS 'Shippings of products';

-- Indexes for foreign keys
CREATE INDEX ix_shipping_product_id ON shipping (product_id);
CREATE INDEX ix_run_shop_id ON run (shop_id);
CREATE INDEX ix_product_shop_id ON product (shop_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX ix_product_shop_id;
DROP INDEX ix_run_shop_id;
DROP INDEX ix_shipping_product_id;

DROP TABLE shipping;
DROP TABLE product;
DROP TABLE run;
DROP TABLE shop;

-- +goose StatementEnd