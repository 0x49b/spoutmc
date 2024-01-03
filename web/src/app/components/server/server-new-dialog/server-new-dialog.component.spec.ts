import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ServerNewDialogComponent } from './server-new-dialog.component';

describe('ServerNewDialogComponent', () => {
  let component: ServerNewDialogComponent;
  let fixture: ComponentFixture<ServerNewDialogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ServerNewDialogComponent]
    })
    .compileComponents();
    
    fixture = TestBed.createComponent(ServerNewDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
