layout: landing
title: EdgeX 项目文档
description: EdgeX 项目的完整文档
---

<section class="hero-section">
  <div class="shell shell--wide">
    <div class="hero-banner">
      <div class="hero-grid">
        <div class="hero-copy">
          <div class="eyebrow">Industrial Edge Documentation</div>
          <h1>EdgeX 文档站，直接进入可落地的工程入口。</h1>
          <p>把 API、部署、驱动、边缘计算和运维资料集中在一个真正可浏览的 Landing Page 里。首页负责“快速进入”，文档页负责“专注阅读”，两种页面采用不同宽度策略。</p>
          <div class="hero-actions">
            <a class="button-link button-link--primary" href="API/API_Index_CN.html">查看 API 文档</a>
            <a class="button-link button-link--secondary" href="man/USER_MANUAL.html">打开用户手册</a>
            <a class="button-link button-link--secondary" href="ARCHITECTURE_V2.html">浏览架构设计</a>
          </div>
          <div class="hero-metrics">
            <div class="metric-card">
              <strong>10+</strong>
              <span>核心文档分区</span>
            </div>
            <div class="metric-card">
              <strong>V2.0.1</strong>
              <span>当前公开版本</span>
            </div>
            <div class="metric-card">
              <strong>2026-04-09</strong>
              <span>首页信息更新时间</span>
            </div>
          </div>
        </div>
        <aside class="hero-panel">
          <p class="hero-panel__label">Start Here</p>
          <ul>
            <li>先看 API 索引，快速了解平台能力边界与接口组织。</li>
            <li>再看用户手册与快速启动，把部署、联调和交付路径串起来。</li>
            <li>需要深入时，继续进入驱动、边缘计算、南北向数据与运维专题。</li>
          </ul>
        </aside>
      </div>
    </div>
  </div>
</section>

<section class="landing-section">
  <div class="shell shell--wide">
    <div class="section-heading">
      <div>
        <div class="section-kicker">Primary Entry</div>
        <h2>首页卡片区改成大屏 2-3 列布局</h2>
        <p>这里保留首页该有的“入口感”。在大屏上使用 3 列卡片，宽度收缩后自动变为 2 列或 1 列，方便快速扫视主要文档域。</p>
      </div>
    </div>

    <div class="cards-grid">
      <article class="feature-card">
        <span class="feature-card__tag">API</span>
        <h3>接口与能力总览</h3>
        <p>先从 API 索引进入，涵盖认证、通道设备管理、边缘计算、北向配置和系统管理。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="API/API_Index_CN.html">API 索引中文</a>
          <a class="mini-link" href="API/API_Index.html">API Index</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">Manual</span>
        <h3>用户手册与场景实践</h3>
        <p>适合实施、交付和现场调试阶段，快速串联系统使用、边缘流和最佳实践。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="man/USER_MANUAL.html">用户手册</a>
          <a class="mini-link" href="man/EDGE_COMPUTING_BEST_PRACTICES.html">最佳实践</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">Architecture</span>
        <h3>架构与设计</h3>
        <p>聚焦三级架构、后端重构、状态机和数据源/动作设计，适合做方案评审和版本规划。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="ARCHITECTURE_V2.html">架构 V2</a>
          <a class="mini-link" href="STATE_MACHINE_API.html">状态机 API</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">Drivers</span>
        <h3>设备驱动专题</h3>
        <p>覆盖 BACnet、OPC UA、Modbus 的设计、测试、优化和故障分析，方便按协议深入。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="BACnet_设计说明.html">BACnet</a>
          <a class="mini-link" href="OPC_UA_Design.html">OPC UA</a>
          <a class="mini-link" href="MODBUS_OPTIMIZATION.html">Modbus</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">Delivery</span>
        <h3>部署、集成与测试</h3>
        <p>把集成指南、快速启动、验收测试和验证报告放在一起，便于团队走完整交付链路。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="INTEGRATION_GUIDE.html">集成指南</a>
          <a class="mini-link" href="QUICK_REFERENCE.html">快速参考</a>
          <a class="mini-link" href="VERIFICATION_REPORT.html">验证报告</a>
        </div>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">Operations</span>
        <h3>系统管理与运维</h3>
        <p>面向运行期稳定性，覆盖系统设置、网络、认证、数据库集成和 BACnet 运维手册。</p>
        <div class="feature-card__links">
          <a class="mini-link" href="边缘网关系统设置.html">系统设置</a>
          <a class="mini-link" href="auth.html">认证</a>
          <a class="mini-link" href="运维手册_BACnet.html">运维手册</a>
        </div>
      </article>
    </div>
  </div>
</section>

<section class="landing-section">
  <div class="shell shell--wide">
    <div class="section-heading">
      <div>
        <div class="section-kicker">Deep Dive</div>
        <h2>按专题进入更深的文档流</h2>
        <p>第二屏继续承担“分流”职责，但不再像旧首页那样堆满单列链接，而是用更宽的专题卡片承接重点内容。</p>
      </div>
    </div>

    <div class="cards-grid cards-grid--two">
      <article class="feature-card">
        <span class="feature-card__tag">Edge Computing</span>
        <h3>边缘计算</h3>
        <p>从基础功能到高阶能力，再到首页监控、功能走查、逻辑图和拓扑图，适合做产品与技术对齐。</p>
        <ul>
          <li><a href="边缘计算基础功能.html">边缘计算基础功能</a></li>
          <li><a href="边缘计算高阶功能.html">边缘计算高阶功能</a></li>
          <li><a href="边缘计算首页监控.html">边缘计算首页监控</a></li>
          <li><a href="edge_compute_topology_diagram.html">边缘计算拓扑图</a></li>
        </ul>
      </article>

      <article class="feature-card">
        <span class="feature-card__tag">Southbound & Northbound</span>
        <h3>南北向数据</h3>
        <p>聚焦采集质量、通道决策、回归验证，以及 MQTT 数据上下行格式，适合联调和平台对接阶段。</p>
        <ul>
          <li><a href="南向通道指标监控.html">南向通道指标监控</a></li>
          <li><a href="南向采集数据通道质量优化.html">通道质量优化</a></li>
          <li><a href="南向采集通道回归验证测试方案.html">回归验证测试方案</a></li>
          <li><a href="MQTT数据上下行格式.html">MQTT 数据上下行格式</a></li>
        </ul>
      </article>
    </div>
  </div>
</section>

<section class="landing-section">
  <div class="shell shell--wide">
    <div class="wide-panel">
      <div class="section-kicker">Quick Links</div>
      <h2>常用入口</h2>
      <p>如果你只是想快速跳到一个常用页面，可以直接从这里进入。</p>
      <div class="quick-links">
        <a href="API/Authentication_CN.html">认证 API</a>
        <a href="API/Channel_Device_Management_CN.html">通道设备管理</a>
        <a href="API/Edge_Computing_CN.html">边缘计算 API</a>
        <a href="API/Northbound_Configuration_CN.html">北向配置 API</a>
        <a href="API/System_Management_CN.html">系统管理 API</a>
        <a href="PROJECT_COMPLETION_REPORT.html">项目完成报告</a>
        <a href="PROJECT_DELIVERY.html">项目交付</a>
        <a href="quality_score_rules.html">质量评分规则</a>
      </div>
    </div>
  </div>
</section>
