// GIVEN and fully working: a test double for sending email, so tests never
// touch real infrastructure.

export interface SentEmail {
  to: string;
  subject: string;
  body: string;
}

export interface EmailSender {
  send(to: string, subject: string, body: string): void;
}

export class FakeEmailSender implements EmailSender {
  public readonly sent: SentEmail[] = [];

  send(to: string, subject: string, body: string): void {
    this.sent.push({ to, subject, body });
  }
}
