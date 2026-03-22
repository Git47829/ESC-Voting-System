INSERT INTO Voting_Status (VotingID, isOpen, lastChange) VALUES
    (1, TRUE, '12:00:00');

INSERT INTO Land (ID, Name, POT) VALUES
    ('DE', 'Germany', 10),
    ('SE', 'Sweden', 12),
    ('FR', 'France', 8),
    ('ES', 'Spain', 6);

INSERT INTO Komponist (Vorname, Name) VALUES
    ('"Nena"', 'Gabrielle Susanne Kerner'),
    ('Thomas', 'G:son'),
    ('Aria', 'Vidal');

INSERT INTO Kuenstler (Vorname, Name, Typ, Land_ID) VALUES
    ('"Nena"', 'Gabrielle Susanne Kerner', 'solo', 'DE'),
    ('Alice', 'Lindgren', 'duo', 'SE'),
    ('Jean', 'Dupont', 'solo', 'FR');

INSERT INTO Song (ID, Name, Land_ID, Kuenstler_ID, PublikumsPunkte, JuryPunkte, YoutubeURL) VALUES
    (1, 'Irgendwie, Irgendwo, Irgendwann', 'DE', 1, 120, 105, 'https://www.youtube.com/embed/oMHLkcc9I9c'),
    (2, 'Northern Lights', 'SE', 2, 110, 118, 'https://www.youtube.com/embed/Pfo-8z86x80'),
    (3, 'Parisian Nights', 'FR', 3, 98, 112, NULL);

INSERT INTO Song_Komponist (Song_ID, Komponist_ID) VALUES
    (1, 1),
    (1, 2),
    (2, 2),
    (3, 3);
