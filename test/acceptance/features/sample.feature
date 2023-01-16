Feature: Sample Feature

    Background:
        Given Namespace "test" is used
    @smoke
    Scenario: Test namespace
        When Namespace "test" exists
        Then Namespace "test" is deleted
