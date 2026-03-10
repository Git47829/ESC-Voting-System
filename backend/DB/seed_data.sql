INSERT INTO Voting_Status (VotingID, isOpen, lastChange) VALUES
    (1, TRUE, '12:00:00');

INSERT INTO Land (ID, Name, POT) VALUES
    ('DEU', 'Germany', 10),
    ('SWE', 'Sweden', 12),
    ('FRA', 'France', 8),
    ('ESP', 'Spain', 6);

INSERT INTO Komponist (Vorname, Name) VALUES
    ('Lena', 'Meyer-Landrut'),
    ('Thomas', 'G:son'),
    ('Aria', 'Vidal');

INSERT INTO Kuenstler (Vorname, Name, Typ, Land_ID) VALUES
    ('Lena', 'Meyer-Landrut', 'solo', 'DEU'),
    ('Alice', 'Lindgren', 'duo', 'SWE'),
    ('Jean', 'Dupont', 'solo', 'FRA');

INSERT INTO Song (ID, Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte, YoutubeURL) VALUES
    (1, 'Satellite Reprise', 'DEU', 1, 120, 105, 'https://www.youtube.com/embed/Pfo-8z86x80'),
    (2, 'Northern Lights', 'SWE', 2, 110, 118, 'https://www.youtube.com/embed/NAB4NRFgxmg'),
    (3, 'Parisian Nights', 'FRA', 3, 98, 112, NULL);

INSERT INTO Song_Komponist (Song_ID, Komponist_ID) VALUES
    (1, 1),
    (1, 2),
    (2, 2),
    (3, 3);
