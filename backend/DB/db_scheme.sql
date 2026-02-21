-- ================================================
-- Eurovision Song Contest – Datenbankschema
-- ================================================

-- Land
CREATE TABLE Land (
    ID      CHAR(3)      NOT NULL,
    Name    VARCHAR(100) NOT NULL,
    POT     TINYINT UNSIGNED,
    PRIMARY KEY (ID),
    UNIQUE  (Name)
);

-- Komponist
CREATE TABLE Komponist (
    ID      INT          NOT NULL AUTO_INCREMENT,
    Vorname VARCHAR(100) NOT NULL,
    Name    VARCHAR(100) NOT NULL,
    PRIMARY KEY (ID)
);

-- Künstler
CREATE TABLE Kuenstler (
    ID       INT          NOT NULL AUTO_INCREMENT,
    Vorname  VARCHAR(100),
    Name     VARCHAR(200) NOT NULL,
    Typ      ENUM('solo', 'duo', 'gruppe') NOT NULL DEFAULT 'solo',
    Land_ID  CHAR(3)      NOT NULL,           -- Relation: Künstler tritt für Land an
    PRIMARY KEY (ID),
    CONSTRAINT fk_kuenstler_land FOREIGN KEY (Land_ID) REFERENCES Land(ID)
);

-- Song
CREATE TABLE Song (
    ID              INT           NOT NULL AUTO_INCREMENT,
    Name            VARCHAR(200)  NOT NULL,
    Land_ID         CHAR(3)       NOT NULL,   -- Relation: Song gehört zu Land
    Kuenstler_ID    INT           NOT NULL,   -- Relation: Song wird von Künstler gespielt
    PublikumsPunkte SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    JuryPunkte      SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    GesamtPunkte    SMALLINT UNSIGNED GENERATED ALWAYS AS (PublikumsPunkte + JuryPunkte) STORED,
    PRIMARY KEY (ID),
    CONSTRAINT fk_song_land      FOREIGN KEY (Land_ID)      REFERENCES Land(ID),
    CONSTRAINT fk_song_kuenstler FOREIGN KEY (Kuenstler_ID) REFERENCES Kuenstler(ID)
);

-- Beziehung: Song <-> Komponist (n:m)
CREATE TABLE Song_Komponist (
    Song_ID      INT NOT NULL,
    Komponist_ID INT NOT NULL,
    PRIMARY KEY (Song_ID, Komponist_ID),
    CONSTRAINT fk_sk_song      FOREIGN KEY (Song_ID)      REFERENCES Song(ID),
    CONSTRAINT fk_sk_komponist FOREIGN KEY (Komponist_ID) REFERENCES Komponist(ID)
);
