-- Eurovision Song Contest 2026 - Vienna, Austria
-- 35 participating countries | Semi-finals: 12 & 14 May | Grand Final: 16 May 2026
-- YouTube links use embed format (https://www.youtube.com/embed/<videoId>)
-- Note: Denmark, Latvia, San Marino links are national final performances (no separate official MV released)
--       Luxembourg link is the official visualizer

SET NAMES utf8mb4;

-- ============================================================
-- Clear existing entry data (order respects foreign keys)
-- ============================================================
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE Song_Komponist;
TRUNCATE TABLE Song;
TRUNCATE TABLE Kuenstler;
TRUNCATE TABLE Land;
SET FOREIGN_KEY_CHECKS = 1;

-- Reset Voting_Status to closed
DELETE FROM Voting_Status;
INSERT INTO Voting_Status (VotingID, isOpen) VALUES (1, FALSE);

-- ============================================================
-- Land (Countries) — 35 entries, POT nullable
-- ============================================================
INSERT INTO Land (ID, Name, POT) VALUES
    ('AL', 'Albania',        NULL),
    ('AM', 'Armenia',        NULL),
    ('AU', 'Australia',      NULL),
    ('AT', 'Austria',        NULL),
    ('AZ', 'Azerbaijan',     NULL),
    ('BE', 'Belgium',        NULL),
    ('BG', 'Bulgaria',       NULL),
    ('HR', 'Croatia',        NULL),
    ('CY', 'Cyprus',         NULL),
    ('CZ', 'Czechia',        NULL),
    ('DK', 'Denmark',        NULL),
    ('EE', 'Estonia',        NULL),
    ('FI', 'Finland',        NULL),
    ('FR', 'France',         NULL),
    ('GE', 'Georgia',        NULL),
    ('DE', 'Germany',        NULL),
    ('GR', 'Greece',         NULL),
    ('IL', 'Israel',         NULL),
    ('IT', 'Italy',          NULL),
    ('LV', 'Latvia',         NULL),
    ('LT', 'Lithuania',      NULL),
    ('LU', 'Luxembourg',     NULL),
    ('MT', 'Malta',          NULL),
    ('MD', 'Moldova',        NULL),
    ('ME', 'Montenegro',     NULL),
    ('NO', 'Norway',         NULL),
    ('PL', 'Poland',         NULL),
    ('PT', 'Portugal',       NULL),
    ('RO', 'Romania',        NULL),
    ('SM', 'San Marino',     NULL),
    ('RS', 'Serbia',         NULL),
    ('SE', 'Sweden',         NULL),
    ('CH', 'Switzerland',    NULL),
    ('UA', 'Ukraine',        NULL),
    ('GB', 'United Kingdom', NULL);

-- ============================================================
-- Kuenstler (Artists) — explicit IDs for Song FK references
-- ============================================================
INSERT INTO Kuenstler (ID, Vorname, Name, Typ, Land_ID) VALUES
    ( 1, NULL,               'Alis',                           'solo',   'AL'),
    ( 2, NULL,               'Simón',                          'solo',   'AM'),
    ( 3, 'Delta',            'Goodrem',                        'solo',   'AU'),
    ( 4, NULL,               'Cosmó',                          'solo',   'AT'),
    ( 5, NULL,               'Jiva',                           'solo',   'AZ'),
    ( 6, NULL,               'Essyla',                         'solo',   'BE'),
    ( 7, NULL,               'Dara',                           'solo',   'BG'),
    ( 8, NULL,               'Lelek',                          'gruppe', 'HR'),
    ( 9, NULL,               'Antigoni',                       'solo',   'CY'),
    (10, 'Daniel',           'Žižka',                          'solo',   'CZ'),
    (11, 'Søren Torpegaard', 'Lund',                           'solo',   'DK'),
    (12, NULL,               'Vanilla Ninja',                  'gruppe', 'EE'),
    (13, NULL,               'Linda Lampenius & Pete Parkkonen','duo',   'FI'),
    (14, NULL,               'Monroe',                         'solo',   'FR'),
    (15, NULL,               'Bzikebi',                        'gruppe', 'GE'),
    (16, 'Sarah',            'Engels',                         'solo',   'DE'),
    (17, NULL,               'Akylas',                         'solo',   'GR'),
    (18, 'Noam',             'Bettan',                         'solo',   'IL'),
    (19, 'Sal',              'Da Vinci',                       'solo',   'IT'),
    (20, NULL,               'Atvara',                         'gruppe', 'LV'),
    (21, 'Lion',             'Ceccah',                         'solo',   'LT'),
    (22, 'Eva',              'Marija',                         'solo',   'LU'),
    (23, NULL,               'Aidan',                          'solo',   'MT'),
    (24, NULL,               'Satoshi',                        'solo',   'MD'),
    (25, 'Tamara',           'Živković',                       'solo',   'ME'),
    (26, 'Jonas',            'Lovv',                           'solo',   'NO'),
    (27, NULL,               'Alicja',                         'solo',   'PL'),
    (28, NULL,               'Bandidos do Cante',              'gruppe', 'PT'),
    (29, 'Alexandra',        'Căpitănescu',                    'solo',   'RO'),
    (30, NULL,               'Senhit',                         'solo',   'SM'),
    (31, NULL,               'Lavina',                         'gruppe', 'RS'),
    (32, NULL,               'Felicia',                        'solo',   'SE'),
    (33, 'Veronica',         'Fusaro',                         'solo',   'CH'),
    (34, NULL,               'Leléka',                         'gruppe', 'UA'),
    (35, NULL,               'Look Mum No Computer',           'gruppe', 'GB');

-- ============================================================
-- Song — all points start at 0; YoutubeURL in embed format
-- ============================================================
INSERT INTO Song (ID, Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte, YoutubeURL) VALUES
    ( 1, 'Nân',                   'AL',  1, 0, 0, 'https://www.youtube.com/embed/b9AdRrA554o'),
    ( 2, 'Paloma Rumba',          'AM',  2, 0, 0, 'https://www.youtube.com/embed/5EXoK-lgocw'),
    ( 3, 'Eclipse',               'AU',  3, 0, 0, 'https://www.youtube.com/embed/EUMCr1pnaMY'),
    ( 4, 'Tanzschein',            'AT',  4, 0, 0, 'https://www.youtube.com/embed/zPGP9ZphxiY'),
    ( 5, 'Just Go',               'AZ',  5, 0, 0, 'https://www.youtube.com/embed/iMDBPe25JhM'),
    ( 6, 'Dancing on the Ice',    'BE',  6, 0, 0, 'https://www.youtube.com/embed/9sfI4g6DWTU'),
    ( 7, 'Bangaranga',            'BG',  7, 0, 0, 'https://www.youtube.com/embed/J3oGYo_mekw'),
    ( 8, 'Andromeda',             'HR',  8, 0, 0, 'https://www.youtube.com/embed/vl7Jqnw10sU'),
    ( 9, 'Jalla',                 'CY',  9, 0, 0, 'https://www.youtube.com/embed/TzSs51BiQrE'),
    (10, 'Crossroads',            'CZ', 10, 0, 0, 'https://www.youtube.com/embed/6ea25aRGpLo'),
    (11, 'Før vi går hjem',       'DK', 11, 0, 0, 'https://www.youtube.com/embed/xKzEP9dwoss'),
    (12, 'Too Epic to Be True',   'EE', 12, 0, 0, 'https://www.youtube.com/embed/Nd8iB7acrZ4'),
    (13, 'Liekinheitin',          'FI', 13, 0, 0, 'https://www.youtube.com/embed/9bfwNIYb96Q'),
    (14, 'Regarde !',             'FR', 14, 0, 0, 'https://www.youtube.com/embed/ujoCYrvvTYQ'),
    (15, 'On Replay',             'GE', 15, 0, 0, 'https://www.youtube.com/embed/coh-lygCINY'),
    (16, 'Fire',                  'DE', 16, 0, 0, 'https://www.youtube.com/embed/AzvRc3eH_rA'),
    (17, 'Ferto',                 'GR', 17, 0, 0, 'https://www.youtube.com/embed/NGwNTd_DA9s'),
    (18, 'Michelle',              'IL', 18, 0, 0, 'https://www.youtube.com/embed/xWCnWSoG8nI'),
    (19, 'Per sempre sì',         'IT', 19, 0, 0, 'https://www.youtube.com/embed/V406FAGkhyQ'),
    (20, 'Ēnā',                   'LV', 20, 0, 0, 'https://www.youtube.com/embed/6C2ivaB5D00'),
    (21, 'Sólo quiero más',       'LT', 21, 0, 0, 'https://www.youtube.com/embed/0H-PXnbhG7A'),
    (22, 'Mother Nature',         'LU', 22, 0, 0, 'https://www.youtube.com/embed/bXIOlWnDzaY'),
    (23, 'Bella',                 'MT', 23, 0, 0, 'https://www.youtube.com/embed/E_g9gGBIZhM'),
    (24, 'Viva, Moldova!',        'MD', 24, 0, 0, 'https://www.youtube.com/embed/SViojHjNSzc'),
    (25, 'Nova zora',             'ME', 25, 0, 0, 'https://www.youtube.com/embed/nuvy2d60HbI'),
    (26, 'Ya Ya Ya',              'NO', 26, 0, 0, 'https://www.youtube.com/embed/MasllzWk_bQ'),
    (27, 'Pray',                  'PL', 27, 0, 0, 'https://www.youtube.com/embed/q78cnYIoF9Y'),
    (28, 'Rosa',                  'PT', 28, 0, 0, 'https://www.youtube.com/embed/jyHaE6GqaaQ'),
    (29, 'Choke Me',              'RO', 29, 0, 0, 'https://www.youtube.com/embed/yn0YmI0dPb8'),
    (30, 'Superstar',             'SM', 30, 0, 0, 'https://www.youtube.com/embed/wOQe-fQSFxg'),
    (31, 'Kraj mene',             'RS', 31, 0, 0, 'https://www.youtube.com/embed/FJTLKBOOE98'),
    (32, 'My System',             'SE', 32, 0, 0, 'https://www.youtube.com/embed/ibbfS8iG450'),
    (33, 'Alice',                 'CH', 33, 0, 0, 'https://www.youtube.com/embed/PfpYGAzW5dM'),
    (34, 'Ridnym',                'UA', 34, 0, 0, 'https://www.youtube.com/embed/SoEXezpblAc'),
    (35, 'Eins, Zwei, Drei',      'GB', 35, 0, 0, 'https://www.youtube.com/embed/niMKvJ-Itq8');
