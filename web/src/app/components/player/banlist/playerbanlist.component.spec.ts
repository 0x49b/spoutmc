import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PlayerbanlistComponent } from './playerbanlist.component';

describe('PlayerbanlistComponent', () => {
  let component: PlayerbanlistComponent;
  let fixture: ComponentFixture<PlayerbanlistComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PlayerbanlistComponent]
    })
    .compileComponents();
    
    fixture = TestBed.createComponent(PlayerbanlistComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
