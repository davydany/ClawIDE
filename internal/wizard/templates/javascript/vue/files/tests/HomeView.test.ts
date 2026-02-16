import { mount } from '@vue/test-utils';
import { describe, it, expect } from 'vitest';
import HomeView from '../src/views/HomeView.vue';

describe('HomeView', () => {
  it('renders the welcome heading', () => {
    const wrapper = mount(HomeView);

    expect(wrapper.text()).toContain('Welcome');
  });
});
