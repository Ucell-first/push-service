module.exports = {
    extends: ["@commitlint/config-conventional"],
    rules: {
        "type-enum": [
            2,
            "always",
            [
                "feat",     // yangi feature
                "fix",      // bug fix
                "docs",     // dokumentatsiya
                "style",    // kod formatlash
                "refactor", // kod refaktoring
                "test",     // test qo'shish
                "chore",    // boshqa ishlar
                "perf",     // performance
                "ci",       // CI/CD
                "build",    // build jarayoni
                "revert"    // commit qaytarish
            ]
        ],
        "subject-case": [2, "always", "lower-case"],
        "subject-max-length": [2, "always", 50]
    }
};
