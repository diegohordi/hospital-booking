CREATE TABLE tb_user
(
    id       BIGSERIAL    NOT NULL,
    uuid     UUID         NOT NULL,
    email    VARCHAR(250) NOT NULL,
    password VARCHAR(250) NOT NULL,
    role     VARCHAR(50)  NOT NULL,
    CONSTRAINT tb_user_id_pk PRIMARY KEY (id),
    CONSTRAINT tb_user_uuid_uk UNIQUE (uuid),
    CONSTRAINT tb_user_email_uk UNIQUE (email)
);

CREATE TABLE tb_patient
(
    id           BIGSERIAL    NOT NULL,
    uuid         UUID         NOT NULL,
    user_id      BIGINT       NOT NULL,
    name         VARCHAR(250) NOT NULL,
    email        VARCHAR(250) NOT NULL,
    mobile_phone VARCHAR(12),
    CONSTRAINT tb_patient_id_pk PRIMARY KEY (id),
    CONSTRAINT tb_patient_uuid_uk UNIQUE (uuid),
    CONSTRAINT tb_patient_email_uk UNIQUE (email),
    CONSTRAINT tb_patient_user_id_fk FOREIGN KEY (user_id) REFERENCES tb_user (id)
);

CREATE TABLE tb_doctor
(
    id           BIGSERIAL    NOT NULL,
    uuid         UUID         NOT NULL,
    user_id      BIGINT       NOT NULL,
    name         VARCHAR(250) NOT NULL,
    email        VARCHAR(250) NOT NULL,
    mobile_phone VARCHAR(12),
    specialty    VARCHAR(259),
    CONSTRAINT tb_doctor_id_pk PRIMARY KEY (id),
    CONSTRAINT tb_doctor_uuid_uk UNIQUE (uuid),
    CONSTRAINT tb_doctor_email_uk UNIQUE (email),
    CONSTRAINT tb_doctor_user_id_fk FOREIGN KEY (user_id) REFERENCES tb_user (id)
);

CREATE TABLE tb_block_period
(
    id          BIGSERIAL NOT NULL,
    uuid        UUID      NOT NULL,
    doctor_id   BIGINT    NOT NULL,
    start_date  TIMESTAMP NOT NULL,
    end_date    TIMESTAMP NOT NULL,
    description VARCHAR(250),
    CONSTRAINT tb_block_period_id_pk PRIMARY KEY (id),
    CONSTRAINT tb_block_period_uuid_uk UNIQUE (uuid),
    CONSTRAINT tb_block_period_doctor_id_fk FOREIGN KEY (doctor_id) REFERENCES tb_doctor (id)
);

CREATE TABLE tb_appointment
(
    id         BIGSERIAL NOT NULL,
    uuid       UUID      NOT NULL,
    doctor_id  BIGINT    NOT NULL,
    patient_id BIGINT    NOT NULL,
    date       TIMESTAMP NOT NULL,
    CONSTRAINT tb_appointment_id_pk PRIMARY KEY (id),
    CONSTRAINT tb_appointment_uuid_uk UNIQUE (uuid),
    CONSTRAINT tb_appointment_doctor_id_fk FOREIGN KEY (doctor_id) REFERENCES tb_doctor (id),
    CONSTRAINT tb_appointment_patient_id_fk FOREIGN KEY (patient_id) REFERENCES tb_doctor (id)
);


-- Seeding users
INSERT INTO tb_user (uuid, email, password, role) VALUES
('9f1aab10-dc04-4ab5-9911-87da9b6a9c76', 'patient@hospital.com', '$2a$10$7FvC9T3y/ert5hkuRj37TuQGXPASbBRh1sYJDNRSCfHMqsoJ.4Lgy', 'PATIENT'),
('f5ec116d-7ed6-4c3c-850a-cbd91b203381', 'doctor@hospital.com', '$2a$10$mgvh1tur98fACPDMtKNao.KrdxdXRCttfmLn9QDnehpXpZ1vRaAZG', 'DOCTOR');

-- Seeding patients
INSERT INTO tb_patient (uuid, user_id, name, email, mobile_phone)
SELECT '672b8ea1-5b09-4974-b97b-afb623648789', u.id, 'John Doe', 'patient@hospital.com', '351123123123'
FROM tb_user u WHERE u.uuid = '9f1aab10-dc04-4ab5-9911-87da9b6a9c76';

-- Seeding doctors
INSERT INTO tb_doctor (uuid, user_id, name, email, mobile_phone, specialty)
SELECT '293691a7-9d90-47f9-a502-ff196f9d50e0', u.id, 'Doe John', 'doctor@hospital.com', '351351351351', 'Cardiologist'
FROM tb_user u WHERE u.uuid = 'f5ec116d-7ed6-4c3c-850a-cbd91b203381';