# Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project uses [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Department management (#10): any signed-in user can list and read
  departments, and admins can create, update and delete them. Codes can't
  change, a department still in use can't be deleted, and every change is
  audited. Migration 011 seeds the current MITT B.E. departments (CS, AD,
  AI, CV, EC, ME).

### Fixed
- Roles count only while they are in effect: an ended or future-dated role
  no longer grants permissions at login, refresh or HOD assignment.
- Approving a user with a department scope now checks the department
  exists, and can't race with deleting that department (#9).
- Two admins creating the same department code at once get 409, not 500.

### Changed
- The access request form and USN validator use `AI` for CSE (AI and ML)
  instead of the unconfirmed `CI` code.

## Before this changelog

Phase 0 (foundation) and Phase 1 (identity and access: request access,
approval, activation email, login with rotating refresh tokens, scoped
RBAC, profiles) shipped before the changelog was started. See PR #6 and
the git history.
