export const requireRole = (role) => {
    return (req, res, next) => {
        const currentRole = req.session.role;
        if (!currentRole) {
            res.status(401).json({ error: "Authentication required" });
            return;
        }
        if (currentRole !== role && currentRole !== "admin") {
            res.status(403).json({ error: "Forbidden" });
            return;
        }
        next();
    };
};
