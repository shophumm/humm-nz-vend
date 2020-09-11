create database vend;
use vend;

DROP TABLE IF EXISTS `humm_vend_map`;
--
create table humm_vend_map (
    id int NOT NULL  auto_increment,
    fxl_register_id varchar(255) NOT NULL COMMENT 'i.e humm Device ID',
    fxl_seller_id varchar(255) NOT NULL COMMENT 'i.e Merchant ID in humm',
    fxl_device_signing_key varchar(255) COMMENT 'i.e Device specific signing key allocated by CreateKey',
    origin_domain varchar(255) NOT NULL COMMENT 'Vend origin provided in the initial request',
    vend_register_id varchar(255) NOT NULL COMMENT 'Unique Register ID from Vend',
    created_date datetime DEFAULT CURRENT_TIMESTAMP,
    created_by text NOT NULL ,
    modified_date datetime,
    modified_by text,
    primary key(id)
     
) engine=InnoDB;

CREATE OR REPLACE UNIQUE INDEX unique_registration USING HASH
ON humm_vend_map (vend_register_id, fxl_seller_id, origin_domain);

-- insert test records

-- INSERT INTO humm_vend_map (
--     fxl_register_id,
--     fxl_seller_id,
--     fxl_device_signing_key, 
--     origin_domain,
--     vend_register_id,
--     created_by
-- ) VALUES (
--     'Oxipos',
--     '30188105',
--     'JCjbPGtuniWr',
--     'https://sandbox.humm.com.au',
--     '57d863b4-4ae0-492c-b44a-326db76f7dac',
--     'andrewm'
-- );

DROP TABLE IF EXISTS `sessions`; 
CREATE TABLE sessions (
	id INT NOT NULL AUTO_INCREMENT,
	session_data LONGBLOB,
    created_on TIMESTAMP DEFAULT NOW(),
	modified_on TIMESTAMP NOT NULL DEFAULT NOW() ON UPDATE CURRENT_TIMESTAMP,
    expires_on TIMESTAMP DEFAULT NOW(),
     PRIMARY KEY(`id`)
 ) engine=InnoDB, COMMENT = 'This stores http sessions and is required by the session store handler';
